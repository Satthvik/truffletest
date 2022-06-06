package git

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/kylelemons/godebug/pretty"
	log "github.com/sirupsen/logrus"
	"github.com/trufflesecurity/trufflehog/v3/pkg/common"
	"github.com/trufflesecurity/trufflehog/v3/pkg/pb/credentialspb"
	"github.com/trufflesecurity/trufflehog/v3/pkg/pb/source_metadatapb"
	"github.com/trufflesecurity/trufflehog/v3/pkg/pb/sourcespb"
	"github.com/trufflesecurity/trufflehog/v3/pkg/sources"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestSource_Scan(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	secret, err := common.GetTestSecret(ctx)
	if err != nil {
		t.Fatal(fmt.Errorf("failed to access secret: %v", err))
	}
	basicUser := secret.MustGetField("GITLAB_USER")
	basicPass := secret.MustGetField("GITLAB_PASS")

	type init struct {
		name        string
		verify      bool
		connection  *sourcespb.Git
		concurrency int
	}
	tests := []struct {
		name      string
		init      init
		wantChunk *sources.Chunk
		wantErr   bool
	}{
		{
			name: "local repo",
			init: init{
				name: "this repo",
				connection: &sourcespb.Git{
					Directories: []string{"../../../"},
					Credential: &sourcespb.Git_Unauthenticated{
						Unauthenticated: &credentialspb.Unauthenticated{},
					},
				},
				concurrency: 4,
			},
			wantChunk: &sources.Chunk{
				SourceType: sourcespb.SourceType_SOURCE_TYPE_GIT,
				SourceName: "this repo",
				Verify:     false,
			},
			wantErr: false,
		},
		{
			name: "remote repo, unauthenticated",
			init: init{
				name: "test source",
				connection: &sourcespb.Git{
					Repositories: []string{"https://github.com/dustin-decker/secretsandstuff.git"},
					Credential: &sourcespb.Git_Unauthenticated{
						Unauthenticated: &credentialspb.Unauthenticated{},
					},
				},
				concurrency: 4,
			},
			wantChunk: &sources.Chunk{
				SourceType: sourcespb.SourceType_SOURCE_TYPE_GIT,
				SourceName: "test source",
				Verify:     false,
			},
			wantErr: false,
		},
		{
			name: "remote repo, unauthenticated, concurrency 0",
			init: init{
				name: "test source",
				connection: &sourcespb.Git{
					Repositories: []string{"https://github.com/dustin-decker/secretsandstuff.git"},
					Credential: &sourcespb.Git_Unauthenticated{
						Unauthenticated: &credentialspb.Unauthenticated{},
					},
				},
				concurrency: 0,
			},
			wantChunk: &sources.Chunk{
				SourceType: sourcespb.SourceType_SOURCE_TYPE_GIT,
				SourceName: "test source",
				Verify:     false,
			},
			wantErr: false,
		},
		{
			name: "remote repo, basic auth",
			init: init{
				name: "test source",
				connection: &sourcespb.Git{
					Repositories: []string{"https://github.com/dustin-decker/secretsandstuff.git"},
					Credential: &sourcespb.Git_BasicAuth{
						BasicAuth: &credentialspb.BasicAuth{
							Username: basicUser,
							Password: basicPass,
						},
					},
				},
				concurrency: 4,
			},
			wantChunk: &sources.Chunk{
				SourceType: sourcespb.SourceType_SOURCE_TYPE_GIT,
				SourceName: "test source",
				Verify:     false,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Source{}
			log.SetLevel(log.DebugLevel)

			conn, err := anypb.New(tt.init.connection)
			if err != nil {
				t.Fatal(err)
			}

			err = s.Init(ctx, tt.init.name, 0, 0, tt.init.verify, conn, tt.init.concurrency)
			if (err != nil) != tt.wantErr {
				t.Errorf("Source.Init() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			chunksCh := make(chan *sources.Chunk, 1)
			go func() {
				s.Chunks(ctx, chunksCh)
			}()
			gotChunk := <-chunksCh
			gotChunk.Data = nil
			// Commits don't come in a deterministic order, so remove metadata comparison
			gotChunk.SourceMetadata = nil
			if diff := pretty.Compare(gotChunk, tt.wantChunk); diff != "" {
				t.Errorf("Source.Chunks() %s diff: (-got +want)\n%s", tt.name, diff)
				t.Errorf("Data: %s", string(gotChunk.Data))
			}
		})
	}
}

func Test_generateLink(t *testing.T) {
	type args struct {
		repo   string
		commit string
		file   string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "test link gen",
			args: args{
				repo:   "https://github.com/trufflesec-julian/confluence-go-api.git",
				commit: "047b4a2ba42fc5b6c0bd535c5307434a666db5ec",
				file:   ".gitignore",
			},
			want: "https://github.com/trufflesec-julian/confluence-go-api/blob/047b4a2ba42fc5b6c0bd535c5307434a666db5ec/.gitignore",
		},
		{
			name: "test link gen - no file",
			args: args{
				repo:   "https://github.com/trufflesec-julian/confluence-go-api.git",
				commit: "047b4a2ba42fc5b6c0bd535c5307434a666db5ec",
			},
			want: "https://github.com/trufflesec-julian/confluence-go-api/commit/047b4a2ba42fc5b6c0bd535c5307434a666db5ec",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GenerateLink(tt.args.repo, tt.args.commit, tt.args.file); got != tt.want {
				t.Errorf("generateLink() = %v, want %v", got, tt.want)
			}
		})
	}
}

// We ran into an issue where upgrading a dependency caused the git patch chunking to break
// So this test exists to make sure that when something changes, we know about it.
func TestSource_Chunks_Integration(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	type init struct {
		name       string
		verify     bool
		connection *sourcespb.Git
	}

	type byteCompare struct {
		B     []byte
		Found bool
		Multi bool
	}
	tests := []struct {
		name string
		init init
		//verified
		expectedChunkData map[string]*byteCompare
	}{
		{
			name: "remote repo, unauthenticated",
			init: init{
				name: "test source",
				connection: &sourcespb.Git{
					Repositories: []string{"https://github.com/dustin-decker/secretsandstuff.git"},
					Credential: &sourcespb.Git_Unauthenticated{
						Unauthenticated: &credentialspb.Unauthenticated{},
					},
				},
			},
			expectedChunkData: map[string]*byteCompare{
				"70001020fab32b1fcf2f1f0e5c66424eae649826-aws":   {B: []byte("[default]\naws_access_key_id = AKIAXYZDQCEN4B6JSJQI\naws_secret_access_key = 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\noutput = json\nregion = us-east-2\n")},
				"a6f8aa55736d4a85be31a0048a4607396898647a-bump":  {B: []byte("f\n")},
				"07d96d011005fe8296bdd237c13a06a72e96783d-bump":  {B: []byte(" s \n")},
				"2f251b8c1e72135a375b659951097ec7749d4af9-bump":  {B: []byte(" \n")},
				"e6c8bbabd8796ea3cd85bfc2e55b27e0a491747f-bump":  {B: []byte("oops \n")},
				"735b52b0eb40610002bb1088e902bd61824eb305-bump":  {B: []byte("oops\n")},
				"ce62d79908803153ef6e145e042d3e80488ef747-bump":  {B: []byte("\n")},
				"27fbead3bf883cdb7de9d7825ed401f28f9398f1-slack": {B: []byte("yup, just did that\n\ngithub_lol: \"ffc7e0f9400fb6300167009e42d2f842cd7956e2\"\n\noh, goodness. there's another one!")},
				"8afb0ecd4998b1179e428db5ebbcdc8221214432-slack": {B: []byte("oops might drop a slack token here\n\ngithub_secret=\"369963c1434c377428ca8531fbc46c0c43d037a0\"\n\nyup, just did that"), Multi: true},
				"8fe6f04ef1839e3fc54b5147e3d0e0b7ab971bd5-aws":   {B: []byte("blah blaj\n\nthis is the secret: AKIA2E0A8F3B244C9986\n\nokay thank you bye"), Multi: true},
				"90c75f884c65dc3638ca1610bd9844e668f213c2-aws":   {B: []byte("this is the secret: [Default]\nAccess key Id: AKIAILE3JG6KMS3HZGCA\nSecret Access Key: 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\n"), Multi: true},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Source{}
			log.SetLevel(log.DebugLevel)

			conn, err := anypb.New(tt.init.connection)
			if err != nil {
				t.Fatal(err)
			}
			err = s.Init(ctx, tt.init.name, 0, 0, tt.init.verify, conn, 4)
			if err != nil {
				t.Fatal(err)
			}
			chunksCh := make(chan *sources.Chunk, 1)
			go func() {
				defer close(chunksCh)
				err := s.Chunks(ctx, chunksCh)
				if err != nil {
					panic(err)
				}
			}()

			for chunk := range chunksCh {

				key := ""
				switch meta := chunk.SourceMetadata.GetData().(type) {
				case *source_metadatapb.MetaData_Git:
					key = meta.Git.Commit + "-" + meta.Git.File
				}

				if expectedData, exists := tt.expectedChunkData[key]; !exists {
					t.Errorf("A chunk exists that was not expected with key %q", key)
				} else {
					if bytes.Equal(chunk.Data, expectedData.B) {
						(*tt.expectedChunkData[key]).Found = true
					} else if !expectedData.Multi {
						t.Errorf("Got %q: %q, which was not expected", key, string(chunk.Data))
					}
				}
			}

			for key, expected := range tt.expectedChunkData {
				if !expected.Found {
					t.Errorf("Expected data with key %q not found", key)
				}

			}
		})
	}
}

func TestSource_Chunks_Edge_Cases(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	secret, err := common.GetTestSecret(ctx)
	if err != nil {
		t.Fatal(fmt.Errorf("failed to access secret: %v", err))
	}
	basicUser := secret.MustGetField("GITLAB_USER")
	basicPass := secret.MustGetField("GITLAB_PASS")

	type init struct {
		name       string
		verify     bool
		connection *sourcespb.Git
	}
	tests := []struct {
		name    string
		init    init
		wantErr string
	}{
		{
			name: "empty repo",
			init: init{
				name: "test source",
				connection: &sourcespb.Git{
					Repositories: []string{"https://github.com/git-fixtures/empty.git"},
					Credential: &sourcespb.Git_Unauthenticated{
						Unauthenticated: &credentialspb.Unauthenticated{},
					},
				},
			},
			wantErr: "remote",
		},
		{
			name: "no repo",
			init: init{
				name: "test source",
				connection: &sourcespb.Git{
					Repositories: []string{""},
					Credential: &sourcespb.Git_Unauthenticated{
						Unauthenticated: &credentialspb.Unauthenticated{},
					},
				},
			},
			wantErr: "remote",
		},
		{
			name: "no repo, basic auth",
			init: init{
				name: "test source",
				connection: &sourcespb.Git{
					Repositories: []string{""},
					Credential: &sourcespb.Git_BasicAuth{
						BasicAuth: &credentialspb.BasicAuth{
							Username: basicUser,
							Password: basicPass,
						},
					},
				},
			},
			wantErr: "remote",
		},
		{
			name: "symlinks repo",
			init: init{
				name: "test source",
				connection: &sourcespb.Git{
					Repositories: []string{"https://github.com/git-fixtures/symlinks.git"},
					Credential: &sourcespb.Git_Unauthenticated{
						Unauthenticated: &credentialspb.Unauthenticated{},
					},
				},
			},
		},
		{
			name: "submodule repo",
			init: init{
				name: "test source",
				connection: &sourcespb.Git{
					Repositories: []string{"https://github.com/git-fixtures/submodule.git"},
					Credential: &sourcespb.Git_Unauthenticated{
						Unauthenticated: &credentialspb.Unauthenticated{},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Source{}
			log.SetLevel(log.DebugLevel)

			conn, err := anypb.New(tt.init.connection)
			if err != nil {
				t.Fatal(err)
			}

			err = s.Init(ctx, tt.init.name, 0, 0, tt.init.verify, conn, 4)
			if err != nil {
				t.Errorf("Source.Init() error = %v", err)
				return
			}
			chunksCh := make(chan *sources.Chunk, 1)
			go func() {
				for chunk := range chunksCh {
					chunk.Data = nil
				}

			}()
			if err := s.Chunks(ctx, chunksCh); err != nil && !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Source.Chunks() error = %v, wantErr %v", err, tt.wantErr)
			}

		})
	}
}

func TestPrepareRepo(t *testing.T) {
	tests := []struct {
		uri    string
		path   bool
		remote bool
		err    error
	}{
		{
			uri:    "https://github.com/dustin-decker/secretsandstuff.git",
			path:   true,
			remote: true,
			err:    nil,
		},
		{
			uri:    "http://github.com/dustin-decker/secretsandstuff.git",
			path:   true,
			remote: true,
			err:    nil,
		},
		{
			uri:    "file:///path/to/file.json",
			path:   true,
			remote: false,
			err:    nil,
		},
		{
			uri:    "no bueno",
			path:   false,
			remote: false,
			err:    fmt.Errorf("unsupported Git URI: no bueno"),
		},
	}

	for _, tt := range tests {
		repo, b, err := PrepareRepo(tt.uri)
		var repoLen bool
		if len(repo) > 0 {
			repoLen = true
		} else {
			repoLen = false
		}
		if repoLen != tt.path || b != tt.remote {
			t.Errorf("PrepareRepo(%v) got: %v, %v, %v want: %v, %v, %v", tt.uri, repo, b, err, tt.path, tt.remote, tt.err)
		}
	}
}

func BenchmarkPrepareRepo(b *testing.B) {
	uri := "https://github.com/dustin-decker/secretsandstuff.git"
	for i := 0; i < b.N; i++ {
		_, _, _ = PrepareRepo(uri)
	}
}
