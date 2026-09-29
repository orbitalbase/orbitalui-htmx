# orbitalui-htmx

Reusable server-rendered UI components for Go applications using templ, HTMX, Alpine.js, and Tailwind CSS.

## Components

Import `github.com/orbitalbase/orbitalui-htmx/components` and compose the templ components in your own page templates. The package includes text and number fields, select controls, text areas, character counts, inline icons, chat message bubbles, suggestion chips, assistant avatars, and loading indicators.

The first public component set is:

- `HTMXPostForm` for composing child components inside an HTMX POST form.
- `TextInputField`, `NumberInputControl`, `DropdownField`, `CompactDropdown`, `TextAreaControl`, and `CharacterCount` for data entry.
- `ChatThread`, `SummaryCard`, `SummaryField`, `Icon`, `AssistantAvatar`, `AssistantMessage`, `UserMessage`, `SuggestionChip`, and `LoadingSpinner` for chat and feedback surfaces.
- `SelectOption` for select menu data.

The field components accept CSS classes and Alpine model expressions from the caller. Import `styles/components.css` in your app stylesheet for the optional `oui-field-control` and `oui-field-label` base styles. The library does not choose HTMX endpoints or application state for you; wrap the controls in your own HTMX forms.

## Local development

```sh
go tool templ generate
```

Applications using Tailwind should add the library's `.templ` files to Tailwind's source scan so utility classes used by library components are included in the output CSS.
