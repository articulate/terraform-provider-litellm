package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestTeamReadAcceptsOnlyRoundTrippableLegacyBooleanMetadataProtocol(t *testing.T) {
	for _, test := range []struct {
		name   string
		key    string
		prior  string
		remote bool
		ok     bool
	}{
		{"guardrails false", "disable_global_guardrails", "false", false, true},
		{"guardrails true", "disable_global_guardrails", "true", true, false},
		{"guardrails changed to true", "disable_global_guardrails", "false", true, false},
		{"other key false", "other_flag", "false", false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			const id = "legacy-boolean-team"
			const semantic = `{"allowed_passthrough_routes":["/model-disco"]}`
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				writer.Header().Set("Content-Type", "application/json")
				if request.Method != http.MethodGet || request.URL.Query().Get("team_id") != id {
					http.Error(writer, "unexpected request", http.StatusBadRequest)
					return
				}
				switch request.URL.Path {
				case "/team/info":
					_ = json.NewEncoder(writer).Encode(map[string]interface{}{
						"team_id": id, "keys": []interface{}{}, "team_memberships": []interface{}{},
						"team_info": map[string]interface{}{
							"team_id": id, "team_alias": id,
							"metadata": map[string]interface{}{
								"allowed_passthrough_routes": []string{"/model-disco"},
								test.key:                     test.remote,
								"unowned_boolean":            true,
							},
						},
					})
				case "/team/permissions_list":
					_ = json.NewEncoder(writer).Encode(map[string]interface{}{"team_id": id, "team_member_permissions": []interface{}{}})
				default:
					http.NotFound(writer, request)
				}
			}))
			defer server.Close()
			protocolServer, schemas := configuredImportProtocolServer(t, ctx, server.URL)
			const typeName = "litellm_team"
			schema := schemas.ResourceSchemas[typeName]
			state := teamSemanticProtocolValue(t, schema, map[string]interface{}{
				"id": id, "team_id": id, "team_alias": id, "metadata_json": semantic,
				"metadata": map[string]tftypes.Value{
					test.key: tftypes.NewValue(tftypes.String, test.prior),
				},
			})
			prepared, err := prepareTeamSemanticDictionary(ctx, types.StringValue(semantic), types.MapNull(types.StringType))
			if err != nil {
				t.Fatal(err)
			}
			provenance, err := encodeSemanticDictionaryProvenance(ctx, prepared.provenance)
			if err != nil {
				t.Fatal(err)
			}
			private, err := json.Marshal(map[string][]byte{teamMetadataJSONProvenancePrivateKey: provenance})
			if err != nil {
				t.Fatal(err)
			}
			read, err := protocolServer.ReadResource(ctx, &tfprotov6.ReadResourceRequest{TypeName: typeName, CurrentState: state, Private: private})
			if err != nil {
				t.Fatal(err)
			}
			if !test.ok {
				if !accessGroupProtocolDiagnosticsHaveError(read.Diagnostics) {
					t.Fatal("refresh accepted a boolean that a legacy write would send back as a string")
				}
				return
			}
			if accessGroupProtocolDiagnosticsHaveError(read.Diagnostics) {
				t.Fatalf("refresh: diagnostics=%s", agentProtocolDiagnosticsText(read.Diagnostics))
			}
			attributes := protocolAttributeMap(t, schema, read.NewState)
			var metadata map[string]tftypes.Value
			if err := attributes["metadata"].As(&metadata); err != nil {
				t.Fatal(err)
			}
			if len(metadata) != 1 || protocolString(t, metadata[test.key]) != "false" {
				t.Fatalf("legacy metadata=%v, want only %s=\"false\"", metadata, test.key)
			}
			if got := protocolString(t, attributes["metadata_json"]); got != semantic {
				t.Fatalf("metadata_json=%q, want %q", got, semantic)
			}
		})
	}
}
