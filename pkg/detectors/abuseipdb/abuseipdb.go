package abuseipdb

import (
	"context"
	"io/ioutil"

	// "log"
	"regexp"
	"strings"

	"net/http"

	"github.com/trufflesecurity/trufflehog/v3/pkg/common"
	"github.com/trufflesecurity/trufflehog/v3/pkg/detectors"
	"github.com/trufflesecurity/trufflehog/v3/pkg/pb/detectorspb"
)

type Scanner struct{}

// Ensure the Scanner satisfies the interface at compile time.
var _ detectors.Detector = (*Scanner)(nil)

var (
	client = common.SaneHttpClient()

	// Make sure that your group is surrounded in boundary characters such as below to reduce false positives.
	keyPat = regexp.MustCompile(detectors.PrefixRegex([]string{"abuseipdb"}) + `\b([a-z0-9]{80})\b`)
)

// Keywords are used for efficiently pre-filtering chunks.
// Use identifiers in the secret preferably, or the provider name.
func (s Scanner) Keywords() []string {
	return []string{"abuseipdb"}
}

// FromData will find and optionally verify AbuseIPDB secrets in a given set of bytes.
func (s Scanner) FromData(ctx context.Context, verify bool, data []byte) (results []detectors.Result, err error) {
	dataStr := string(data)

	matches := keyPat.FindAllStringSubmatch(dataStr, -1)

	for _, match := range matches {
		if len(match) != 2 {
			continue
		}
		resMatch := strings.TrimSpace(match[1])

		s1 := detectors.Result{
			DetectorType: detectorspb.DetectorType_AbuseIPDB,
			Raw:          []byte(resMatch),
		}

		if verify {
			req, err := http.NewRequestWithContext(ctx, "GET", "https://api.abuseipdb.com/api/v2/check?ipAddress=118.25.6.39", nil)
			if err != nil {
				continue
			}
			req.Header.Add("Key", resMatch)
			res, err := client.Do(req)
			if err == nil {
				bodyBytes, err := ioutil.ReadAll(res.Body)
				if err == nil {
					bodyString := string(bodyBytes)
					validResponse := strings.Contains(bodyString, `ipAddress`)
					//errCode := strings.Contains(bodyString, `AbuseIPDB APIv2 Server.`)

					defer res.Body.Close()
					if res.StatusCode >= 200 && res.StatusCode < 300 {
						if validResponse {
							s1.Verified = true
						} else {
							s1.Verified = false
						}
					} else {
						// This function will check false positives for common test words, but also it will make sure the key appears 'random' enough to be a real key.
						if detectors.IsKnownFalsePositive(resMatch, detectors.DefaultFalsePositives, true) {
							continue
						}
					}
				}
			}
		}
		results = append(results, s1)
	}

	return detectors.CleanResults(results), nil
}
