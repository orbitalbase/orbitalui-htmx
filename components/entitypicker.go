// EntityPicker types + Alpine config (entitypicker.templ) — the v2
// EntityPicker combobox semantics ported to Alpine: a text input that
// autocompletes against a JSON endpoint, carrying the chosen record's id in
// a hidden form field (docs/ui-v1-plan.md §3, docs/ui-components.md §3b).
package components

import (
	"encoding/json"

	"github.com/a-h/templ"
)

// EntityPickerOpts configures the combobox.
type EntityPickerOpts struct {
	ID    string // unique per-instance id (also the input/label ids)
	Label string // accessible name ("Account", "Party", …)
	Name  string // form field name — carries the chosen record id

	// Endpoint is the JSON autocomplete URL for one entity domain,
	// including its book scoping — e.g. "/api/accounts?book=ACME". The
	// picker appends "&q=<query>" and expects
	// {"items":[{"id":"…","label":"…","sublabel":"…"}]}
	// (the wire contract — docs/ui-components.md §3b). Registered per
	// domain by that domain's phase.
	Endpoint templ.SafeURL

	Placeholder string // default "Search…"
	AllowClear  bool   // shows a clear (×) when a value is picked
	Required    bool   // server-side semantics only (the UI flags it)

	// Value/ValueLabel restore server state after a failed submit: the
	// picked record id + the label to display in the input.
	Value      string
	ValueLabel string

	// LabelName, when set, renders a SECOND hidden input carrying the picked
	// record's human label instead of its id. A form that stores the label
	// itself needs it: the bank import wizard records the statement's account
	// NUMBER as text (bank_import.bank_account_no existed for that, and the
	// importer resolves the number to the entity), so posting a uuid would make
	// the field meaningless. Empty omits the input entirely, so an id-only
	// picker ships exactly the markup it shipped before this existed.
	LabelName string

	Class string // extra classes on the wrapper

	// Create opts the picker into "create in place" (Airtable/Linear): a
	// query with no existing match grows a synthetic row and Enter posts a
	// new record without leaving the field. NIL IS THE DEFAULT and must
	// stay behaviourally identical to the plain combobox — every call site
	// that does not set it keeps its current markup byte for byte.
	Create *EntityPickerCreateOpts
}

// EntityPickerCreateOpts is the opt-in half of EntityPickerOpts: everything
// the picker needs to mint a record it could not find.
//
// The endpoint's contract is fixed and owned by the consuming domain (the
// same domain that registers Endpoint): POST the typed text, get back the
// picker's own wire shape {"id","label","sublabel"} on 2xx. The picker
// deliberately knows nothing else about the record — no field list, no
// book scoping — because the domain already owns the URL via Endpoint.
//
//	templ.SafeURL("/api/parties?book=ACME") // GET  — search
//	templ.SafeURL("/api/parties?book=ACME") // POST — create
//
// Returns 201 (or 200) + {"id":"41","label":"Acme Freight","sublabel":""}.
type EntityPickerCreateOpts struct {
	// Endpoint is the POST target. It carries the domain's book scoping the
	// same way Endpoint does — the picker appends nothing.
	Endpoint templ.SafeURL

	// Field is the JSON body key carrying the typed text ("name",
	// "display_name", …). Empty falls back to "name" client-side, so a
	// domain with the obvious field can leave it out.
	Field string

	// Noun is the human noun for the row text — "bank account" renders
	// Create bank account "abc123". Empty renders the bare
	// Create "abc123", which is the right thing when Endpoint and Label
	// already say what is being made.
	Noun             string
	RequireBookScope bool
}

// pickerConfig is the JSON blob handed to the Alpine factory (app.js
// LedgerUI.picker). Kept as a struct so the wire is typed + marshalled once.
type pickerConfig struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Name        string `json:"name"`
	Endpoint    string `json:"endpoint"`
	Placeholder string `json:"placeholder"`
	AllowClear  bool   `json:"allowClear"`
	Required    bool   `json:"required"`
	Value       string `json:"value"`
	ValueLabel  string `json:"valueLabel"`
	LabelName   string `json:"labelName,omitempty"`

	// Omitted entirely when nil (omitempty on a pointer), so a picker
	// without Create ships the exact JSON this config shipped before the
	// feature existed — the JS truthiness check is a second line of
	// defence, not the first.
	Create *pickerCreateConfig `json:"create,omitempty"`
}

// pickerCreateConfig is the create sub-object of the Alpine config. Split
// from EntityPickerCreateOpts so the wire shape is one struct literal and the
// Go API can keep templ.SafeURL out of the JSON.
type pickerCreateConfig struct {
	Endpoint         string `json:"endpoint"`
	Field            string `json:"field"`
	Noun             string `json:"noun"`
	RequireBookScope bool   `json:"requireBookScope,omitempty"`
}

// pickerDataExpr renders the Alpine x-data expression for opts:
// "LedgerUI.picker({...})". The JSON is HTML-escaped by templ in the
// attribute, so quotes inside labels are safe.
func pickerDataExpr(o EntityPickerOpts) string {
	cfg := pickerConfig{
		ID:          o.ID,
		Label:       o.Label,
		Name:        o.Name,
		Endpoint:    string(o.Endpoint),
		Placeholder: o.Placeholder,
		AllowClear:  o.AllowClear,
		Required:    o.Required,
		Value:       o.Value,
		ValueLabel:  o.ValueLabel,
		LabelName:   o.LabelName,
	}
	// A Create with no endpoint would arm the row and then POST nowhere, so
	// it is dropped rather than rendered — a config error must not become a
	// dead button. pickerCreates is the same predicate the markup uses, so
	// the JSON and the row can never disagree about whether create is on.
	if pickerCreates(o) {
		cfg.Create = &pickerCreateConfig{
			Endpoint:         string(o.Create.Endpoint),
			Field:            o.Create.Field,
			Noun:             o.Create.Noun,
			RequireBookScope: o.Create.RequireBookScope,
		}
	}
	b, err := json.Marshal(cfg)
	if err != nil {
		// Cannot happen for a plain struct; degrade to an empty picker.
		return "LedgerUI.picker({})"
	}
	return "LedgerUI.picker(" + string(b) + ")"
}

// pickerClass joins the wrapper base class + extras.
func pickerClass(o EntityPickerOpts) string {
	if o.Class == "" {
		return "entity-picker"
	}
	return "entity-picker " + o.Class
}

// pickerPlaceholder resolves the input placeholder.
func pickerPlaceholder(o EntityPickerOpts) string {
	if o.Placeholder != "" {
		return o.Placeholder
	}
	return "Search…"
}

// pickerListID is the options list id (aria-controls/activedescendant).
func pickerListID(id string) string { return id + "-list" }
