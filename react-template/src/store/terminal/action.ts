import { createAction } from "@reduxjs/toolkit"

export const command = createAction("command", (input: string) => ({ payload: input }))
export const eCommand = createAction("event_command", (data: string) => ({ payload: data }))
export const eCommandEOF = createAction("event_command_eof", (data: string) => ({ payload: data }))
export const addCommandHistory = createAction("add_command_history", (input: string) => ({ payload: input }))
export const cancelCommand = createAction("cancel_command")
export const openTerminal = createAction("open_terminal", (el: HTMLElement) => ({ payload: el }))
export const closeTerminal = createAction("close_terminal")

export default {
	command,
	cancelCommand,
	openTerminal,
	closeTerminal,
}
