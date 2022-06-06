package okta

import (
	"context"
	"fmt"
	"net/http"
	"regexp"

	"github.com/trufflesecurity/trufflehog/v3/pkg/common"
	"github.com/trufflesecurity/trufflehog/v3/pkg/detectors"
	"github.com/trufflesecurity/trufflehog/v3/pkg/pb/detectorspb"
)

type Scanner struct{}

// Ensure the Scanner satisfies the interface at compile time.
var _ detectors.Detector = (*Scanner)(nil)

var (
	domainPat = regexp.MustCompile(`[a-z0-9-]{1,40}\.okta(?:preview|-emea){0,1}\.com`)
	tokenPat  = regexp.MustCompile(`00[a-zA-Z0-9_-]{40}`)
	// TODO: Oauth client secrets
)

// Keywords are used for efficiently pre-filtering chunks.
// Use identifiers in the secret preferably, or the provider name.
func (s Scanner) Keywords() []string {
	return []string{"okta"}
}

// FromData will find and optionally verify Okta secrets in a given set of bytes.
func (s Scanner) FromData(ctx context.Context, verify bool, data []byte) (results []detectors.Result, err error) {
	for _, tokenMatch := range tokenPat.FindAll(data, -1) {
		token := string(tokenMatch)

		for _, domainMatch := range domainPat.FindAll(data, -1) {
			domain := string(domainMatch)

			s := detectors.Result{
				DetectorType: detectorspb.DetectorType_Okta,
				Raw:          []byte(token),
			}

			if verify {
				// curl -v -X GET \
				// -H "Accept: application/json" \
				// -H "Content-Type: application/json" \
				// -H "Authorization: Bearer token" \
				// "https://subdomain.okta.com/api/v1/groups?limit=1"
				//

				url := fmt.Sprintf("https://%s/api/v1/groups?limit=1", domain)
				req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
				if err != nil {
					return results, err
				}
				req.Header.Set("Accept", "application/json")
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", fmt.Sprintf("SSWS %s", token))

				resp, err := common.SaneHttpClient().Do(req)
				if err != nil {
					return results, err
				}
				defer resp.Body.Close()
				if resp.StatusCode >= 200 && resp.StatusCode < 300 {
					s.Verified = true
				}
			}

			if !s.Verified {
				if detectors.IsKnownFalsePositive(string(s.Raw), detectors.DefaultFalsePositives, true) {
					continue
				}
			}

			results = append(results, s)
		}
	}

	return
}
