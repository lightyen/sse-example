import { createAction } from "@reduxjs/toolkit"

export const shell = createAction("shell", (input: string) => ({ payload: input }))
export const eShell = createAction("event_shell", (data: string) => ({ payload: data }))
export const openTerminal = createAction("open_terminal", (el: HTMLElement) => ({ payload: el }))
export const closeTerminal = createAction("close_terminal")

export default {
	shell,
	openTerminal,
	closeTerminal,
}
