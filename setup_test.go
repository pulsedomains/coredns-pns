package pns

import (
	"testing"

	"github.com/coredns/caddy"
)

func TestPNSParse(t *testing.T) {
	tests := []struct {
		key                string
		inputFileRules     string
		err                string
		connection         string
		plslinknameservers []string
		ipfsgatewayas      []string
		ipfsgatewayaaaas   []string
	}{
		{ // 0
			".",
			`pns {
			}`,
			"Testfile:2 - Error during parsing: no connection",
			"",
			nil,
			nil,
			nil,
		},
		{ // 1
			".",
			`pns {
			   connection
			}`,
			"Testfile:2 - Error during parsing: invalid connection; no value",
			"",
			nil,
			nil,
			nil,
		},
		{ // 2
			".pls.link",
			`pns {
			  connection /home/test/.ethereum/geth.ipc
			  plslinknameservers ns1.plsdns.fyi
			}`,
			"",
			"/home/test/.ethereum/geth.ipc",
			[]string{"ns1.plsdns.fyi."},
			nil,
			nil,
		},
		{ // 3
			".",
			`pns {
			  connection http://localhost:8545/
			  plslinknameservers ns1.plsdns.fyi ns2.plsdns.fyi
			}`,
			"",
			"http://localhost:8545/",
			[]string{"ns1.plsdns.fyi.", "ns2.plsdns.fyi."},
			nil,
			nil,
		},
		{ // 4
			".",
			`pns {
			  connection http://localhost:8545/
			  ipfsgatewaya
			}`,
			"Testfile:3 - Error during parsing: invalid IPFS gateway A; no value",
			"",
			nil,
			nil,
			nil,
		},
		{ // 5
			".",
			`pns {
			  connection http://localhost:8545/
			  plslinknameservers ns1.plsdns.fyi ns2.plsdns.fyi
			  ipfsgatewaya 193.62.81.1
			}`,
			"",
			"",
			nil,
			[]string{"193.62.81.1"},
			nil,
		},
		{ // 6
			".",
			`pns {
			  connection http://localhost:8545/
			  ipfsgatewayaaaa
			}`,
			"Testfile:3 - Error during parsing: invalid IPFS gateway AAAA; no value",
			"",
			nil,
			nil,
			nil,
		},
		{ // 7
			".",
			`pns {
			  connection http://localhost:8545/
			  plslinknameservers ns1.plsdns.fyi ns2.plsdns.fyi
			  ipfsgatewayaaaa fe80::b8fb:325d:fb5a:40e7
			}`,
			"",
			"",
			nil,
			nil,
			[]string{"fe80::b8fb:325d:fb5a:40e7"},
		},
		{ // 8
			"tls://.:8053",
			`pns {
			  connection http://localhost:8545/
			  plslinknameservers ns1.plsdns.fyi ns2.plsdns.fyi
			  ipfsgatewayaaaa fe80::b8fb:325d:fb5a:40e7
			}`,
			"",
			"",
			nil,
			nil,
			[]string{"fe80::b8fb:325d:fb5a:40e7"},
		},
		{ // 9
			".:8053",
			`pns {
			  connection http://localhost:8545/ bad
			  ipfsgatewayaaaa fe80::b8fb:325d:fb5a:40e7
			}`,
			"Testfile:2 - Error during parsing: invalid connection; multiple values",
			"",
			nil,
			nil,
			nil,
		},
		{ // 10
			".:8053",
			`pns {
			  connection http://localhost:8545/
			  plslinknameservers ns1.plsdns.fyi ns2.plsdns.fyi
			  ipfsgatewaya 193.62.81.1 193.62.81.2
			}`,
			"",
			"",
			nil,
			[]string{"193.62.81.1", "193.62.81.2"},
			nil,
		},
		{ // 11
			".:8053",
			`pns {
			  connection http://localhost:8545/
			  plslinknameservers ns1.plsdns.fyi ns2.plsdns.fyi
			  ipfsgatewayaaaa fe80::b8fb:325d:fb5a:40e7 fe80::b8fb:325d:fb5a:40e8
			}`,
			"",
			"",
			nil,
			nil,
			[]string{"fe80::b8fb:325d:fb5a:40e7", "fe80::b8fb:325d:fb5a:40e8"},
		},
		{ // 12
			".:8053",
			`pns {
			  connection http://localhost:8545/
			  ipfsgatewayaaaa fe80::b8fb:325d:fb5a:40e7 fe80::b8fb:325d:fb5a:40e8
			  bad
			}`,
			"Testfile:4 - Error during parsing: unknown value bad",
			"",
			nil,
			nil,
			nil,
		},
	}

	for i, test := range tests {
		c := caddy.NewTestController("pns", test.inputFileRules)
		c.Key = test.key
		connection, plslinknameservers, ipfsgatewayas, ipfsgatewayaaaas, err := pnsParse(c)

		if test.err != "" {
			if err == nil {
				t.Fatalf("Failed to obtain expected error at test %d", i)
			}
			if err.Error() != test.err {
				t.Fatalf("Unexpected error \"%s\" at test %d", err.Error(), i)
			}
		} else {
			if err != nil {
				t.Fatalf("Unexpected error \"%s\" at test %d", err.Error(), i)
			} else {
				if test.connection != "" && connection != test.connection {
					t.Fatalf("Test %d connection expected %v, got %v", i, test.connection, connection)
				}
				if test.plslinknameservers != nil {
					if len(plslinknameservers) != len(test.plslinknameservers) {
						t.Fatalf("Test %d plslinknameservers expected %v entries, got %v", i, len(test.plslinknameservers), len(plslinknameservers))
					}
					for j := range test.plslinknameservers {
						if plslinknameservers[j] != test.plslinknameservers[j] {
							t.Fatalf("Test %d plslinknameservers expected %v, got %v", i, test.plslinknameservers[j], plslinknameservers[j])
						}
					}
				}
				if test.ipfsgatewayas != nil {
					if len(ipfsgatewayas) != len(test.ipfsgatewayas) {
						t.Fatalf("Test %d ipfsgatewayas expected %v entries, got %v", i, len(test.ipfsgatewayas), len(ipfsgatewayas))
					}
					for j := range test.ipfsgatewayas {
						if ipfsgatewayas[j] != test.ipfsgatewayas[j] {
							t.Fatalf("Test %d ipfsgatewayas expected %v, got %v", i, test.ipfsgatewayas[j], ipfsgatewayas[j])
						}
					}
				}
				if test.ipfsgatewayaaaas != nil {
					if len(ipfsgatewayaaaas) != len(test.ipfsgatewayaaaas) {
						t.Fatalf("Test %d ipfsgatewayaaaas expected %v entries, got %v", i, len(test.ipfsgatewayaaaas), len(ipfsgatewayaaaas))
					}
					for j := range test.ipfsgatewayaaaas {
						if ipfsgatewayaaaas[j] != test.ipfsgatewayaaaas[j] {
							t.Fatalf("Test %d ipfsgatewayaaaas expected %v, got %v", i, test.ipfsgatewayaaaas[j], ipfsgatewayaaaas[j])
						}
					}
				}
			}
		}
	}
}
