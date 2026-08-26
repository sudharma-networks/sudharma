package publicrpc

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

type iamPolicy struct {
	Version   string `json:"Version"`
	Statement []struct {
		Effect   string      `json:"Effect"`
		Action   interface{} `json:"Action"`
		Resource interface{} `json:"Resource"`
	} `json:"Statement"`
}

func actionStrings(v interface{}) []string {
	switch x := v.(type) {
	case string:
		return []string{x}
	case []interface{}:
		out := make([]string, 0, len(x))
		for _, item := range x {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func loadPolicy(t *testing.T, name string) iamPolicy {
	t.Helper()
	b, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	var p iamPolicy
	if err := json.Unmarshal(b, &p); err != nil {
		t.Fatalf("parse %s: %v", name, err)
	}
	if p.Version != "2012-10-17" || len(p.Statement) == 0 {
		t.Fatalf("invalid IAM policy envelope in %s", name)
	}
	return p
}

func TestGitHubActionsPolicyIsRestrictedToWalletProxyLambda(t *testing.T) {
	p := loadPolicy(t, "github-actions-testnet-policy.json")
	allowed := map[string]bool{
		"lambda:GetFunction":                 true,
		"lambda:GetFunctionConfiguration":    true,
		"lambda:UpdateFunctionCode":          true,
		"lambda:UpdateFunctionConfiguration": true,
	}
	wantResource := "arn:aws:lambda:ap-south-1:981626123397:function:Sudharma-Testnet-Wallet-Proxy"
	for _, s := range p.Statement {
		if s.Effect != "Allow" {
			t.Fatalf("unexpected effect %q", s.Effect)
		}
		for _, a := range actionStrings(s.Action) {
			if !allowed[a] {
				t.Errorf("unexpected GitHub deployment action %q", a)
			}
			if a == "*" || strings.HasPrefix(strings.ToLower(a), "iam:") {
				t.Errorf("forbidden broad/IAM action %q", a)
			}
		}
		if r, ok := s.Resource.(string); !ok || r != wantResource {
			t.Errorf("GitHub deployment resource = %#v, want exact wallet proxy Lambda ARN", s.Resource)
		}
	}
}

func TestLambdaExecutionPolicyContainsOnlyVpcAndLogActions(t *testing.T) {
	p := loadPolicy(t, "lambda-execution-policy.json")
	allowedPrefixes := []string{"ec2:CreateNetworkInterface", "ec2:DescribeNetworkInterfaces", "ec2:DescribeSubnets", "ec2:DeleteNetworkInterface", "ec2:AssignPrivateIpAddresses", "ec2:UnassignPrivateIpAddresses", "logs:CreateLogGroup", "logs:CreateLogStream", "logs:PutLogEvents"}
	seenENI, seenLogs := false, false
	for _, s := range p.Statement {
		if s.Effect != "Allow" {
			t.Fatalf("unexpected effect %q", s.Effect)
		}
		for _, a := range actionStrings(s.Action) {
			ok := false
			for _, want := range allowedPrefixes {
				if a == want { ok = true; break }
			}
			if !ok {
				t.Errorf("unexpected Lambda execution action %q", a)
			}
			if strings.HasPrefix(a, "ec2:") { seenENI = true }
			if strings.HasPrefix(a, "logs:") { seenLogs = true }
		}
	}
	if !seenENI || !seenLogs {
		t.Fatalf("execution policy must include both VPC ENI and CloudWatch Logs permissions")
	}
}

func TestIamPoliciesContainNoCredentialsOrAdministratorAccess(t *testing.T) {
	for _, name := range []string{"github-actions-testnet-policy.json", "lambda-execution-policy.json"} {
		b, err := os.ReadFile(name)
		if err != nil { t.Fatalf("read %s: %v", name, err) }
		lower := strings.ToLower(string(b))
		for _, forbidden := range []string{"administratoraccess", "accesskey", "secretaccesskey", "iam:createuser", "iam:attachuserpolicy"} {
			if strings.Contains(lower, strings.ToLower(forbidden)) {
				t.Errorf("%s contains forbidden material %q", name, forbidden)
			}
		}
	}
}
