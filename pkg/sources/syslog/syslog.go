package syslog

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/trufflesecurity/trufflehog/v3/pkg/common"
	"io"
	"net"
	"runtime"
	"strconv"
	"time"

	"github.com/bill-rich/go-syslog/pkg/syslogparser/rfc3164"
	"github.com/crewjam/rfc5424"
	"github.com/go-errors/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/trufflesecurity/trufflehog/v3/pkg/pb/source_metadatapb"
	"github.com/trufflesecurity/trufflehog/v3/pkg/pb/sourcespb"
	"github.com/trufflesecurity/trufflehog/v3/pkg/sources"
)

const nilString = ""

type Source struct {
	name     string
	sourceId int64
	jobId    int64
	verify   bool
	syslog   *Syslog
	aCtx     context.Context
	sources.Progress
	conn *sourcespb.Syslog
}

type Syslog struct {
	sourceType         sourcespb.SourceType
	sourceName         string
	sourceID           int64
	jobID              int64
	sourceMetadataFunc func(hostname, appname, procid, timestamp, facility, client string) *source_metadatapb.MetaData
	verify             bool
	concurrency        *semaphore.Weighted
}

func NewSyslog(sourceType sourcespb.SourceType, jobID, sourceID int64, sourceName string, verify bool, concurrency int,
	sourceMetadataFunc func(hostname, appname, procid, timestamp, facility, client string) *source_metadatapb.MetaData,
) *Syslog {
	return &Syslog{
		sourceType:         sourceType,
		sourceName:         sourceName,
		sourceID:           sourceID,
		jobID:              jobID,
		sourceMetadataFunc: sourceMetadataFunc,
		verify:             verify,
		concurrency:        semaphore.NewWeighted(int64(concurrency)),
	}
}

// Ensure the Source satisfies the interface at compile time.
var _ sources.Source = (*Source)(nil)

// Type returns the type of source.
// It is used for matching source types in configuration and job input.
func (s *Source) Type() sourcespb.SourceType {
	return sourcespb.SourceType_SOURCE_TYPE_SYSLOG
}

func (s *Source) SourceID() int64 {
	return s.sourceId
}

func (s *Source) JobID() int64 {
	return s.jobId
}

func (s *Source) InjectConnection(conn *sourcespb.Syslog) {
	s.conn = conn
}

// Init returns an initialized Syslog source.
func (s *Source) Init(aCtx context.Context, name string, jobId, sourceId int64, verify bool, connection *anypb.Any, concurrency int) error {

	s.aCtx = aCtx
	s.name = name
	s.sourceId = sourceId
	s.jobId = jobId
	s.verify = verify

	var conn sourcespb.Syslog
	err := anypb.UnmarshalTo(connection, &conn, proto.UnmarshalOptions{})
	if err != nil {
		return errors.WrapPrefix(err, "error unmarshalling connection", 0)
	}

	s.conn = &conn

	err = s.verifyConnectionConfig()
	if err != nil {
		return errors.WrapPrefix(err, "invalid configuration", 0)
	}

	s.syslog = NewSyslog(s.Type(), s.jobId, s.sourceId, s.name, s.verify, runtime.NumCPU(),
		func(hostname, appname, procID, timestamp, facility, client string) *source_metadatapb.MetaData {
			return &source_metadatapb.MetaData{
				Data: &source_metadatapb.MetaData_Syslog{
					Syslog: &source_metadatapb.Syslog{
						Hostname:  hostname,
						Appname:   appname,
						Procid:    procID,
						Timestamp: timestamp,
						Facility:  facility,
						Client:    client,
					},
				},
			}
		})
	return nil
}

func (s *Source) verifyConnectionConfig() error {
	tlsEnabled := s.conn.TlsCert != nilString || s.conn.TlsKey != nilString
	if s.conn.Protocol == nilString {
		if tlsEnabled {
			s.conn.Protocol = "tcp"
		} else {
			s.conn.Protocol = "udp"
		}
	}

	if s.conn.Protocol == "udp" && tlsEnabled {
		return fmt.Errorf("TLS is not supported over UDP")
	}

	if s.conn.ListenAddress == nilString {
		s.conn.ListenAddress = ":5140"
	}

	if s.conn.Format == nilString {
		s.conn.Format = "rfc3164"
	}
	return nil
}

// Chunks emits chunks of bytes over a channel.
func (s *Source) Chunks(ctx context.Context, chunksChan chan *sources.Chunk) error {
	switch {
	case s.conn.TlsCert != nilString || s.conn.TlsKey != nilString:
		cert, err := tls.X509KeyPair([]byte(s.conn.TlsCert), []byte(s.conn.TlsKey))
		if err != nil {
			return errors.WrapPrefix(err, "could not load key pair", 0)
		}
		cfg := &tls.Config{Certificates: []tls.Certificate{cert}}
		lis, err := tls.Listen(s.conn.Protocol, s.conn.ListenAddress, cfg)
		if err != nil {
			return errors.WrapPrefix(err, "error creating TLS listener", 0)
		}
		defer lis.Close()

		return s.acceptTCPConnections(ctx, lis, chunksChan)
	case s.conn.Protocol == "tcp":
		lis, err := net.Listen(s.conn.Protocol, s.conn.ListenAddress)
		if err != nil {
			return errors.WrapPrefix(err, "error creating TCP listener", 0)
		}
		defer lis.Close()

		return s.acceptTCPConnections(ctx, lis, chunksChan)
	case s.conn.Protocol == "udp":
		lis, err := net.ListenPacket(s.conn.Protocol, s.conn.ListenAddress)
		if err != nil {
			return errors.WrapPrefix(err, "error creating UDP listener", 0)
		}
		err = lis.SetDeadline(time.Now().Add(time.Second))
		if err != nil {
			return errors.WrapPrefix(err, "could not set UDP deadline", 0)
		}
		defer lis.Close()

		return s.acceptUDPConnections(ctx, lis, chunksChan)
	default:
		return fmt.Errorf("unknown connection type")
	}
}

func (s *Source) parseSyslogMetadata(input []byte, remote string) (*source_metadatapb.MetaData, error) {
	var metadata *source_metadatapb.MetaData
	switch s.conn.Format {
	case "rfc5424":
		message := &rfc5424.Message{}
		err := message.UnmarshalBinary(input)
		if err != nil {
			return metadata, errors.WrapPrefix(err, "could not parse syslog as rfc5424", 0)
		}
		metadata = s.syslog.sourceMetadataFunc(message.Hostname, message.AppName, message.ProcessID, message.Timestamp.String(), nilString, remote)
	case "rfc3164":
		parser := rfc3164.NewParser(input)
		err := parser.Parse()
		if err != nil {
			return metadata, errors.WrapPrefix(err, "could not parse syslog as rfc3164", 0)
		}
		data := parser.Dump()
		metadata = s.syslog.sourceMetadataFunc(data["hostname"].(string), nilString, nilString, data["timestamp"].(time.Time).String(), strconv.Itoa(data["facility"].(int)), remote)
	}
	return metadata, nil
}

func (s *Source) monitorConnection(ctx context.Context, conn net.Conn, chunksChan chan *sources.Chunk) {
	for {
		if common.IsDone(ctx) {
			return
		}
		err := conn.SetDeadline(time.Now().Add(time.Second))
		if err != nil {
			logrus.WithError(err).Debug("could not set connection deadline deadline")
		}
		input := make([]byte, 8096)
		remote := conn.RemoteAddr()
		_, err = conn.Read(input)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			continue
		}
		logrus.Trace(string(input))
		metadata, err := s.parseSyslogMetadata(input, remote.String())
		if err != nil {
			logrus.WithError(err).Debug("failed to generate metadata")
		}
		chunksChan <- &sources.Chunk{
			SourceName:     s.syslog.sourceName,
			SourceID:       s.syslog.sourceID,
			SourceType:     s.syslog.sourceType,
			SourceMetadata: metadata,
			Data:           input,
			Verify:         s.verify,
		}
	}
}

func (s *Source) acceptTCPConnections(ctx context.Context, netListener net.Listener, chunksChan chan *sources.Chunk) error {
	for {
		if common.IsDone(ctx) {
			return nil
		}
		conn, err := netListener.Accept()
		if err != nil {
			logrus.WithError(err).Debug("failed to accept TCP connection")
			continue
		}
		go s.monitorConnection(ctx, conn, chunksChan)
	}
}

func (s *Source) acceptUDPConnections(ctx context.Context, netListener net.PacketConn, chunksChan chan *sources.Chunk) error {
	for {
		if common.IsDone(ctx) {
			return nil
		}
		input := make([]byte, 65535)
		_, remote, err := netListener.ReadFrom(input)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			continue
		}
		metadata, err := s.parseSyslogMetadata(input, remote.String())
		if err != nil {
			logrus.WithError(err).Debug("failed to parse metadata")
		}
		chunksChan <- &sources.Chunk{
			SourceName:     s.syslog.sourceName,
			SourceID:       s.syslog.sourceID,
			SourceType:     s.syslog.sourceType,
			SourceMetadata: metadata,
			Data:           input,
			Verify:         s.verify,
		}
	}
}
