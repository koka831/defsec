package formatters

import (
	"bytes"
	"testing"

	"github.com/aquasecurity/defsec/parsers/types"
	"github.com/aquasecurity/defsec/providers"
	"github.com/aquasecurity/defsec/providers/aws/dynamodb"
	"github.com/aquasecurity/defsec/rules"
	"github.com/aquasecurity/defsec/severity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_SARIF(t *testing.T) {
	want := `{
  "version": "2.1.0",
  "$schema": "https://json.schemastore.org/sarif-2.1.0-rtm.5.json",
  "runs": [
    {
      "tool": {
        "driver": {
          "informationUri": "https://tfsec.dev",
          "name": "tfsec",
          "rules": [
            {
              "id": "aws-dynamodb-enable-at-rest-encryption",
              "shortDescription": {
                "text": "summary"
              },
              "helpUri": "https://google.com"
            }
          ]
        }
      },
      "results": [
        {
          "ruleId": "aws-dynamodb-enable-at-rest-encryption",
          "ruleIndex": 0,
          "level": "error",
          "message": {
            "text": "Cluster encryption is not enabled."
          },
          "locations": [
            {
              "physicalLocation": {
                "artifactLocation": {
                  "uri": "test.test"
                },
                "region": {
                  "startLine": 123,
                  "endLine": 123
                }
              }
            }
          ]
        }
      ]
    }
  ]
}`
	buffer := bytes.NewBuffer([]byte{})
	formatter := New().AsSARIF().WithWriter(buffer).Build()
	var results rules.Results
	results.Add("Cluster encryption is not enabled.",
		dynamodb.ServerSideEncryption{
			Metadata: types.NewTestMetadata(),
			Enabled:  types.Bool(false, types.NewTestMetadata()),
		})
	results.SetRule(rules.Rule{
		AVDID:       "AVD-AA-9999",
		LegacyID:    "AAA999",
		ShortCode:   "enable-at-rest-encryption",
		Summary:     "summary",
		Explanation: "explanation",
		Impact:      "impact",
		Resolution:  "resolution",
		Provider:    providers.AWSProvider,
		Service:     "dynamodb",
		Links: []string{
			"https://google.com",
		},
		Severity: severity.High,
	})
	require.NoError(t, formatter.Output(results))
	assert.Equal(t, want, buffer.String())
}
