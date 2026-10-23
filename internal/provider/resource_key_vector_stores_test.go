package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestKeyVectorStoresProtocol(t *testing.T) {
	for _, tc := range []struct {
		name        string
		configured  interface{}
		wantRequest interface{}
		wantState   []string
	}{
		{"retain both knowledge stores", []string{"7NUER936QZ", "GJL4LHFP5U"}, []string{"7NUER936QZ", "GJL4LHFP5U"}, []string{"7NUER936QZ", "GJL4LHFP5U"}},
		{"explicit unrestricted", []string{}, []string{}, []string{}},
		{"inherit gateway defaults", nil, nil, []string{"bedrock-kb"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			var posts int
			remote := interface{}([]string{"bedrock-kb"})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/key/generate", "/key/update":
					posts++
					var body map[string]interface{}
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
					}
					permission, present := body["object_permission"].(map[string]interface{})
					wantRequest := tc.wantRequest
					if r.URL.Path == "/key/update" {
						wantRequest = []string{}
					}
					if wantRequest == nil {
						if present {
							t.Errorf("unconfigured grants changed: %#v", permission)
						}
					} else {
						want, _ := json.Marshal(map[string]interface{}{"vector_stores": wantRequest})
						got, _ := json.Marshal(permission)
						if string(got) != string(want) {
							t.Errorf("permission payload = %s, want %s", got, want)
						}
						remote = permission["vector_stores"]
					}
					_, _ = w.Write([]byte(`{"key":"sk-vector-stores-test"}`))
				case "/key/info":
					_ = json.NewEncoder(w).Encode(map[string]interface{}{"key": "sk-vector-stores-test", "info": map[string]interface{}{
						"object_permission": map[string]interface{}{"vector_stores": remote, "mcp_servers": []string{"existing-mcp"}},
					}})
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			protocol, schemas := configuredImportProtocolServer(t, ctx, server.URL)
			schema := schemas.ResourceSchemas["litellm_key"]
			// A missing input must fail before the typed configuration can discard it.
			found := false
			for _, a := range schema.Block.Attributes {
				if a.Name == "vector_stores" {
					found = true
				}
			}
			if !found {
				t.Fatal("key cannot carry the existing vector-store grants")
			}
			values := map[string]interface{}{"key": "sk-vector-stores-test"}
			if tc.configured != nil {
				elements := make([]tftypes.Value, len(tc.configured.([]string)))
				for i, id := range tc.configured.([]string) {
					elements[i] = tftypes.NewValue(tftypes.String, id)
				}
				values["vector_stores"] = elements
			}
			config := accessGroupProtocolDynamicValue(t, schema, organizationProjectProtocolValue(t, schema, values))
			null := accessGroupProtocolDynamicValue(t, schema, tftypes.NewValue(schema.ValueType(), nil))
			planned, err := protocol.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{TypeName: "litellm_key", Config: config, PriorState: null, ProposedNewState: keySemanticCreateProposed(t, schema, values)})
			if err != nil || accessGroupProtocolDiagnosticsHaveError(planned.Diagnostics) {
				t.Fatalf("plan: %v %v", err, planned.Diagnostics)
			}
			applied, err := protocol.ApplyResourceChange(ctx, &tfprotov6.ApplyResourceChangeRequest{TypeName: "litellm_key", Config: config, PriorState: null, PlannedState: planned.PlannedState, PlannedPrivate: planned.PlannedPrivate})
			if err != nil || accessGroupProtocolDiagnosticsHaveError(applied.Diagnostics) {
				t.Fatalf("create: %v %v", err, applied.Diagnostics)
			}
			attrs := protocolAttributeMap(t, schema, applied.NewState)
			got := keyVectorStoreIDs(t, attrs["vector_stores"])
			if !reflect.DeepEqual(got, tc.wantState) {
				t.Fatalf("stored grants = %#v, want %#v", got, tc.wantState)
			}
			if posts != 1 {
				t.Fatalf("create posts = %d", posts)
			}

			// A refresh must expose changed grants, rather than falsely report parity.
			remote = []string{"changed-remotely"}
			refreshed, err := protocol.ReadResource(ctx, &tfprotov6.ReadResourceRequest{TypeName: "litellm_key", CurrentState: applied.NewState, Private: applied.Private})
			if err != nil || accessGroupProtocolDiagnosticsHaveError(refreshed.Diagnostics) {
				t.Fatalf("refresh: %v %v", err, refreshed.Diagnostics)
			}
			attrs = protocolAttributeMap(t, schema, refreshed.NewState)
			got = keyVectorStoreIDs(t, attrs["vector_stores"])
			if !reflect.DeepEqual(got, []string{"changed-remotely"}) {
				t.Fatalf("drift hidden: %#v", got)
			}

			// Clearing a configured grant must send [], not omit the permission leaf.
			values["vector_stores"] = []tftypes.Value{}
			config = accessGroupProtocolDynamicValue(t, schema, organizationProjectProtocolValue(t, schema, values))
			proposed := organizationProjectProtocolReplace(t, schema, refreshed.NewState, map[string]interface{}{"vector_stores": []tftypes.Value{}})
			planned, err = protocol.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{TypeName: "litellm_key", Config: config, PriorState: refreshed.NewState, ProposedNewState: proposed, PriorPrivate: refreshed.Private})
			if err != nil || accessGroupProtocolDiagnosticsHaveError(planned.Diagnostics) {
				t.Fatalf("update plan: %v %v", err, planned.Diagnostics)
			}
			updated, err := protocol.ApplyResourceChange(ctx, &tfprotov6.ApplyResourceChangeRequest{TypeName: "litellm_key", Config: config, PriorState: refreshed.NewState, PlannedState: planned.PlannedState, PlannedPrivate: planned.PlannedPrivate})
			if err != nil || accessGroupProtocolDiagnosticsHaveError(updated.Diagnostics) {
				t.Fatalf("update: %v %v", err, updated.Diagnostics)
			}
			attrs = protocolAttributeMap(t, schema, updated.NewState)
			if got := keyVectorStoreIDs(t, attrs["vector_stores"]); len(got) != 0 || posts != 2 {
				t.Fatalf("cleared grants = %#v, posts = %d", got, posts)
			}
		})
	}
}

func keyVectorStoreIDs(t *testing.T, value tftypes.Value) []string {
	t.Helper()
	var elements []tftypes.Value
	if err := value.As(&elements); err != nil {
		t.Fatal(err)
	}
	result := make([]string, len(elements))
	for i, element := range elements {
		if err := element.As(&result[i]); err != nil {
			t.Fatal(err)
		}
	}
	return result
}
