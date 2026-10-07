// Command browsercheck serves the interactive components with the shipped
// theme, the orbital-ui.js runtime, Alpine and htmx, plus stub endpoints, so
// scripts/browser-check.mjs can drive them in a real browser. It prints the
// listening URL on stdout and serves until killed.
//
//	go run ./internal/browsercheck -addr 127.0.0.1:0
package main

import (
	"context"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/a-h/templ"

	"github.com/orbitalbase/orbitalui-htmx/assets"
	c "github.com/orbitalbase/orbitalui-htmx/components"
)

//go:embed vendor/*.js
var vendor embed.FS

func main() {
	addr := flag.String("addr", "127.0.0.1:0", "listen address")
	flag.Parse()

	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServerFS(assets.Files)))
	vendorFS, _ := fs.Sub(vendor, "vendor")
	mux.Handle("/vendor/", http.StripPrefix("/vendor/", http.FileServerFS(vendorFS)))
	mux.HandleFunc("GET /{$}", page)
	mux.HandleFunc("POST /dialog", dialogSubmit)
	mux.HandleFunc("GET /drawer", drawerBody)
	mux.HandleFunc("POST /drawer", drawerSubmit)
	mux.HandleFunc("GET /picker", pickerSearch)
	mux.HandleFunc("POST /picker", pickerCreate)
	mux.HandleFunc("GET /swap", swapped)
	mux.HandleFunc("POST /confirm", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("HX-Trigger", "closeDialog")
		io.WriteString(w, `<p id="confirm-result">Archived</p>`)
	})

	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("http://%s/\n", ln.Addr())
	log.Fatal(http.Serve(ln, mux))
}

func render(w http.ResponseWriter, r *http.Request, cs ...templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	for _, x := range cs {
		if err := x.Render(r.Context(), w); err != nil {
			log.Print(err)
		}
	}
}

func raw(s string) templ.Component { return templ.Raw(s) }

func seq(cs ...templ.Component) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		for _, x := range cs {
			if err := x.Render(ctx, w); err != nil {
				return err
			}
		}
		return nil
	})
}

func picker(id string) templ.Component {
	return c.EntityPicker(c.EntityPickerOpts{
		ID: id, Label: "Customer", Name: id, Endpoint: "/picker", AllowClear: true,
		Create: &c.EntityPickerCreateOpts{Endpoint: "/picker", Noun: "customer"},
	})
}

func nameForm(errMsg string) templ.Component {
	return seq(
		raw(`<form id="dialog-form" method="post" action="/dialog" hx-post="/dialog" hx-target="this" hx-swap="outerHTML">`),
		c.LabelledField("Name", "dlg-name", c.FieldOpts{Error: errMsg},
			c.FieldInput(c.FieldInputOpts{ID: "dlg-name", Name: "name", Invalid: errMsg != ""})),
		raw(`<div class="ui-dialog__actions"><button type="submit" class="ui-button ui-button--primary" id="dlg-save">Save</button></div></form>`),
	)
}

func page(w http.ResponseWriter, r *http.Request) {
	render(w, r,
		raw(`<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Orbital UI browser check</title>
<link rel="stylesheet" href="/static/tokens.css"><link rel="stylesheet" href="/static/controls.css"><link rel="stylesheet" href="/static/theme.css">
<script src="/vendor/htmx.min.js" defer></script><script src="/static/orbital-ui.js" defer></script><script src="/vendor/alpine.min.js" defer></script>
</head><body><main>`),
		c.PageHeader("Browser check", c.PageHeaderOpts{}),
		c.Button(c.ButtonVariantPrimary, "", c.ButtonOpts{Label: "Open dialog", Attrs: templ.Attributes{"id": "open-dialog", "data-open-dialog": "dlg"}}),
		c.ModalShell(c.ModalOpts{ID: "dlg", Title: "New contact"}, nameForm("")),
		c.ConfirmButton("Archive", "Archive record", "Archive this record?", "It leaves every register.",
			c.ConfirmOpts{ID: "cfm", Action: "/confirm", HX: true, HXTarget: "#confirm-out", BusyLabel: "Archiving…"}, nil),
		raw(`<div id="confirm-out"></div>`),
		c.Button(c.ButtonVariantSecondary, "", c.ButtonOpts{Label: "Open drawer", Attrs: templ.Attributes{
			"id": "open-drawer", "data-open-drawer": "drw", "hx-get": "/drawer", "hx-target": "#" + c.DrawerContentID("drw"), "hx-swap": "innerHTML",
		}}),
		c.Drawer(c.DrawerOpts{ID: "drw", Title: "Edit template", Fullscreen: true}, nil),
		raw(`<form id="picker-form">`), picker("customer"), raw(`</form>`),
		raw(`<button type="button" id="load-swap" hx-get="/swap" hx-target="#swap-target" hx-swap="innerHTML">Load more</button><div id="swap-target"></div>`),
		raw(`</main></body></html>`),
	)
}

func dialogSubmit(w http.ResponseWriter, r *http.Request) {
	if strings.TrimSpace(r.FormValue("name")) == "" {
		w.WriteHeader(http.StatusUnprocessableEntity)
		render(w, r, nameForm("Enter a name."))
		return
	}
	w.Header().Set("HX-Trigger", "closeDialog")
	render(w, r, nameForm(""))
}

func drawerBody(w http.ResponseWriter, r *http.Request) {
	render(w, r, raw(`<form hx-post="/drawer" hx-target="this" hx-swap="outerHTML"><h2 id="drawer-heading">Template details</h2>
<input class="field-control" name="title" id="drawer-title"><button type="submit" class="ui-button" id="drawer-save">Save</button></form>`))
}

func drawerSubmit(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("HX-Trigger", "drawerClose")
	io.WriteString(w, `<p>Saved</p>`)
}

type pickItem struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Sublabel string `json:"sublabel,omitempty"`
}

var customers = []pickItem{{"c1", "Acme Sdn Bhd", "C-001"}, {"c2", "Northstar Technologies", "C-002"}, {"c3", "Atlas Manufacturing", "C-003"}}

func pickerSearch(w http.ResponseWriter, r *http.Request) {
	q := strings.ToLower(r.URL.Query().Get("q"))
	out := []pickItem{}
	for _, it := range customers {
		if q == "" || strings.Contains(strings.ToLower(it.Label), q) {
			out = append(out, it)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"items": out})
}

func pickerCreate(w http.ResponseWriter, r *http.Request) {
	var body map[string]string
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body["name"] == "" || body["request_key"] == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pickItem{ID: "new-" + body["request_key"][:8], Label: body["name"]})
}

func swapped(w http.ResponseWriter, r *http.Request) {
	render(w, r, raw(`<div id="swapped">`), picker("swapped-picker"), raw(`</div>`))
}
