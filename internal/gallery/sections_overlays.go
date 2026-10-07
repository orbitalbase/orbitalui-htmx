package main

import (
	"github.com/a-h/templ"

	c "github.com/orbitalbase/orbitalui-htmx/components"
)

// Dialogs, drawers and the picker's list are closed or fixed-position at rest.
// The gallery pins them inside bordered stages (g-* classes only; no ui-*
// rule is restyled, only positioning is neutralised) so each can be reviewed
// next to the rest of the page.
const overlaysCSS = `<style>
.g-stage { position: relative; display: grid; place-items: center; min-height: 22rem; padding: 1.5rem; overflow: hidden; background: var(--oui-scrim); border-radius: 12px; isolation: isolate; }
.g-stage .ui-dialog { position: static; margin: 0; animation: none; }
.g-stage--drawer { display: block; height: 34rem; padding: 0; transform: translateZ(0); }
.g-stage--drawer .drawer-overlay { position: absolute; z-index: 1; visibility: visible; opacity: 1; background: transparent; pointer-events: none; transition: none; }
.g-stage--drawer .drawer-panel { transform: none; transition: none; pointer-events: auto; }
.g-stage--drawer .drawer-panel--full { height: 100%; width: 100%; }
.g-open-picker { position: relative; max-width: 22rem; }
.g-open-picker .entity-picker__list { position: static; margin-top: 4px; animation: none; }
</style>`

func init() {
	register(section{ID: "dialogs", Title: "Dialogs", Note: "Native dialog frame (shown inline over a scrim stage), confirm dialog with consequence, form dialog with an inline error.", Body: dialogsBody()})
	register(section{ID: "drawers", Title: "Drawers", Note: "Right slide-over from 40rem up, bottom sheet below; open-state stages. Resize to 390px to review the sheet.", Body: drawersBody()})
	register(section{ID: "picker", Title: "Entity picker", Note: "Closed with a value, empty, and the open list states (options, active row, create row, loading, error, empty) drawn from the same classes.", Body: pickerBody()})
}

func stage(extraClass string, cs ...templ.Component) templ.Component {
	return wrap(`<div class="g-stage `+extraClass+`">`, `</div>`, cs...)
}

func dialogForm(err string) templ.Component {
	return seq(
		raw(`<form onsubmit="return false">`),
		c.LabelledField("Name", "g-dlg-name", c.FieldOpts{Error: err, Help: "As it appears on invoices."},
			c.FieldInput(c.FieldInputOpts{ID: "g-dlg-name", Name: "name", Value: "Acme Sdn Bhd", Invalid: err != ""})),
		raw(`<div class="ui-dialog__actions">`),
		c.Button(c.ButtonVariantSecondary, "", c.ButtonOpts{Label: "Cancel"}),
		c.Button(c.ButtonVariantPrimary, "", c.ButtonOpts{Label: "Save", Type: "submit"}),
		raw(`</div></form>`),
	)
}

func dialogsBody() templ.Component {
	return seq(
		raw(overlaysCSS),
		groupStack("ModalShell: form",
			stage("", c.ModalShell(c.ModalOpts{ID: "g-dlg-form", Title: "New contact", Open: true}, dialogForm(""))),
		),
		groupStack("ModalShell: validation error",
			stage("", c.ModalShell(c.ModalOpts{ID: "g-dlg-err", Title: "New contact", Open: true}, dialogForm("Enter a name."))),
		),
		groupStack("ConfirmButton: the dialog it opens (consequence sentence, then the real submit)",
			stage("", c.ModalShell(c.ModalOpts{ID: "g-cfm-static", Title: "Archive this record?", Open: true, Class: "ui-dialog--confirm"}, seq(
				raw(`<p class="ui-dialog__consequence">It leaves every register and can be restored from Settings.</p><div class="ui-dialog__actions">`),
				c.Button(c.ButtonVariantSecondary, "", c.ButtonOpts{Label: "Cancel"}),
				c.Button(c.ButtonVariantDanger, "", c.ButtonOpts{Label: "Archive record"}),
				raw(`</div>`),
			))),
		),
		groupStack("ConfirmButton: trigger",
			cell("the trigger button on the page", c.ConfirmButton("Archive", "Archive record", "Archive this record?", "It leaves every register and can be restored from Settings.",
				c.ConfirmOpts{ID: "g-cfm", Action: "/confirm"}, nil)),
		),
	)
}

func drawerBody() templ.Component {
	return seq(
		raw(`<p style="margin:0 0 12px;color:var(--oui-text-secondary)">Drawer content sits here: forms, logs and review panels load into this region.</p>`),
		c.LabelledField("Template name", "g-drw-name", c.FieldOpts{}, c.FieldInput(c.FieldInputOpts{ID: "g-drw-name", Name: "t", Value: "Board resolution"})),
		raw(`<div style="display:flex;gap:8px;margin-top:16px">`),
		c.Button(c.ButtonVariantPrimary, "", c.ButtonOpts{Label: "Save"}),
		c.Button(c.ButtonVariantSecondary, "", c.ButtonOpts{Label: "Cancel"}),
		raw(`</div>`),
	)
}

func drawersBody() templ.Component {
	return seq(
		raw(overlaysCSS+`<script>document.addEventListener("DOMContentLoaded",function(){document.querySelectorAll(".g-stage--drawer .drawer-panel").forEach(function(p){p.classList.add("is-open")})})</script>`),
		groupStack("Drawer: title and subtitle",
			stage("g-stage--drawer", c.Drawer(c.DrawerOpts{ID: "g-drw-a", Title: "Edit template", Subtitle: "Changes apply to new documents only."}, drawerBody())),
		),
		groupStack("Drawer: fullscreen toggle",
			stage("g-stage--drawer", c.Drawer(c.DrawerOpts{ID: "g-drw-b", Title: "Reasoning trace", Fullscreen: true}, drawerBody())),
		),
		groupStack("Drawer: tabbed (the host fills the tab strip)",
			stage("g-stage--drawer", c.Drawer(c.DrawerOpts{ID: "g-drw-c", Title: "Open records", Tabbed: true, Fullscreen: true}, drawerBody())),
		),
	)
}

func pickerBody() templ.Component {
	opts := func(id string, o c.EntityPickerOpts) templ.Component {
		o.ID, o.Label, o.Name = id, "Customer", id
		return cell(id, c.EntityPicker(o))
	}
	list := func(inner string) templ.Component {
		return wrap(`<div class="g-open-picker"><div class="entity-picker"><ul class="entity-picker__list" role="listbox">`, `</ul></div></div>`, raw(inner))
	}
	return seq(
		raw(overlaysCSS),
		groupGrid("Closed",
			opts("g-pk-empty", c.EntityPickerOpts{Endpoint: "/e"}),
			opts("g-pk-value", c.EntityPickerOpts{Endpoint: "/e", Value: "c1", ValueLabel: "Acme Sdn Bhd", AllowClear: true}),
			opts("g-pk-create", c.EntityPickerOpts{Endpoint: "/e", Placeholder: "Find or create a customer", Create: &c.EntityPickerCreateOpts{Endpoint: "/e", Noun: "customer"}}),
		),
		groupGrid("Open list (static render of the runtime's rows)",
			cell("options, second row active, first selected", list(`
<li class="entity-picker__option" aria-selected="true" role="option"><button type="button"><span class="entity-picker__label">Acme Sdn Bhd</span><span class="entity-picker__sublabel">C-001</span></button></li>
<li class="entity-picker__option entity-picker__option--active" role="option"><button type="button"><span class="entity-picker__label">Northstar Technologies</span><span class="entity-picker__sublabel">C-002</span></button></li>
<li class="entity-picker__option" role="option"><button type="button"><span class="entity-picker__label">Atlas Manufacturing with a very long registered name that truncates</span><span class="entity-picker__sublabel">C-003</span></button></li>`)),
			cell("with the create row", list(`
<li class="entity-picker__option" role="option"><button type="button"><span class="entity-picker__label">Zeta Holdings</span><span class="entity-picker__sublabel">C-014</span></button></li>
<li class="entity-picker__option entity-picker__option--active" role="option"><button type="button" class="entity-picker__option-btn--create-row"><span class="entity-picker__label">Create customer "Zeta"</span></button></li>`)),
			cell("loading", list(`<li class="entity-picker__state"><span class="ui-button__progress" aria-hidden="true"></span>Loading…</li>`)),
			cell("error", list(`<li class="entity-picker__state entity-picker__state--error" role="alert">Couldn't load options — try again.</li>`)),
			cell("no matches", list(`<li class="entity-picker__state">No matches</li>`)),
		),
	)
}
