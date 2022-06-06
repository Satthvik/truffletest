package paypaloauth

import (
	"context"
	b64 "encoding/base64"
	"fmt"
	"net/http"
	"regexp"
	"strings"

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
	idPat  = regexp.MustCompile(`\b([A-Za-z0-9_\.]{7}-[A-Za-z0-9_\.]{72})\b`)
	keyPat = regexp.MustCompile(`\b([A-Za-z0-9_\.]{69}-[A-Za-z0-9_\.]{10})\b`)
)

// Keywords are used for efficiently pre-filtering chunks.
// Use identifiers in the secret preferably, or the provider name.
func (s Scanner) Keywords() []string {
	return []string{"paypal"}
}

// FromData will find and optionally verify PaypalOauth secrets in a given set of bytes.
func (s Scanner) FromData(ctx context.Context, verify bool, data []byte) (results []detectors.Result, err error) {
	dataStr := string(data)

	matches := keyPat.FindAllStringSubmatch(dataStr, -1)
	idmatches := idPat.FindAllStringSubmatch(dataStr, -1)

	for _, match := range matches {
		if len(match) != 2 {
			continue
		}
		resMatch := strings.TrimSpace(match[1])
		for _, idMatch := range idmatches {
			if len(idMatch) != 2 {
				continue
			}
			residMatch := strings.TrimSpace(idMatch[1])

			s1 := detectors.Result{
				DetectorType: detectorspb.DetectorType_PaypalOauth,
				Raw:          []byte(resMatch),
			}

			if verify {
				data := fmt.Sprintf("%s:%s", residMatch, resMatch)
				encoded := b64.StdEncoding.EncodeToString([]byte(data))
				payload := strings.NewReader("grant_type=client_credentials")
				req, err := http.NewRequestWithContext(ctx, "POST", "https://api-m.sandbox.paypal.com/v1/oauth2/token", payload)
				if err != nil {
					continue
				}
				req.Header.Add("Accept", "application/json")
				req.Header.Add("Accept-Language", "en_US")
				req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
				req.Header.Add("Authorization", fmt.Sprintf("Basic %s", encoded))
				res, err := client.Do(req)
				if err == nil {
					defer res.Body.Close()
					if res.StatusCode >= 200 && res.StatusCode < 300 {
						s1.Verified = true
					} else {
						// This function will check false positives for common test words, but also it will make sure the key appears 'random' enough to be a real key.
						if detectors.IsKnownFalsePositive(resMatch, detectors.DefaultFalsePositives, true) {
							continue
						}
					}
				}
			}

			results = append(results, s1)
		}

	}

	return detectors.CleanResults(results), nil
}
