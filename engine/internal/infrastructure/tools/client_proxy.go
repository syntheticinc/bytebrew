package tools

// ClientOperationsProxy is a marker interface preserved for forward
// compatibility. Historically it carried `AskUserQuestionnaire` for the legacy
// ask_user tool, but ask_user was removed (replaced by show_structured_output
// in form mode, which is non-blocking and emits an event directly via the
// session event stream — no client-side proxying needed). Concrete
// implementations may extend this interface in the future when a new
// client-side proxied tool is introduced.
type ClientOperationsProxy interface{}
