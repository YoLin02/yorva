# YORVA Desktop UI Design Rules

## 1. Baseline

The Desktop Settings page is the visual and interaction baseline for management UI.
New Runtime and Instance surfaces should reuse its restrained visual language:

- flat page background with clear whitespace;
- one page title and one concise explanation;
- compact section headings and setting-style rows;
- light borders and separators instead of nested card stacks;
- 36 px form controls and compact action buttons;
- YORVA orange only for the active tab, primary action and enabled switch;
- no decorative animation, oversized status decoration or redundant headings.

## 2. Resource page versus configuration page

A resource page is an index and status surface. It may show:

- resource groups, counts and short descriptions;
- existing resource rows and authoritative state;
- one clear action per group, such as **Add connection**, **Create Profile** or
  **Configure bindings**;
- simple one-control settings that do not require a workflow, such as selecting the
  Runtime default Profile.

Do not keep appending editable forms to the resource page. A flow must open a dedicated
configuration page when it contains credentials, multiple dependent fields, a list of
choices, multi-Instance selection, validation/test steps, or destructive confirmation.

The configuration page must:

- replace the resource-page body instead of expanding below it;
- begin with a visible back action, one task title and one task explanation;
- group related fields with Settings-style spacing and light separators;
- keep the primary action and cancel/back action together at the bottom;
- return to the resource page after a successful synchronous mutation, or after the
  accepted Operation reaches a successful terminal state;
- keep failure feedback on the configuration page so the user can correct and retry;
- preserve daemon state in TanStack Query rather than copying resources into page state.

Dialogs remain appropriate only for short transient decisions. Long forms and resource
workflows belong on configuration pages.

## 3. Layout and responsive behavior

- The content column must fit the supported compact Desktop window without horizontal
  overflow.
- Tabs must stay on one clean row at the supported minimum window width and must not show
  a decorative scrollbar.
- Two-column forms collapse to one column when space is constrained.
- Long resource names truncate in list rows; configuration values may wrap where reading
  the full value is necessary.
- Empty, loading, error and success states occupy the same page hierarchy as their data;
  they must not create a second competing page title.

## 4. Runtime Models information architecture

The Runtime **Models** resource page contains four setting-style groups:

1. Provider connections: count, existing connections and an **Add connection** entry.
2. Model Profiles: count, existing Profiles and a **Create Profile** entry.
3. Runtime default: one compact Profile selector.
4. Instance bindings: authoritative binding rows and a **Configure bindings** entry.

The following are separate configuration pages:

- Add Provider connection: reviewed Provider, display name and protected credential.
- Create Model Profile: Provider connection, display name, model selection and default
  model.
- Configure Instance bindings: Profile, binding mode and target Instances.

This split changes presentation only. Runtime resources, Operations, credential handling
and authoritative Hermes read-back remain owned by the existing API and application flow.
