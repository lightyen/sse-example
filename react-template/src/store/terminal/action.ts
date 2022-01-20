import { createAction } from "@reduxjs/toolkit"

export const command = createAction("command", (payload: string) => {
	const e = new TextEncoder()
	const arr = e.encode(payload)
	console.log(
		Array.from(arr).map(v => {
			let hex = v.toString(16).toUpperCase()
			if (hex.length < 2) {
				hex = "0" + hex
			}
			return hex
		}),
	)
	return { payload }
})

export const eTerminal = createAction("event_terminal", (data: string) => ({ payload: data }))
export const eTerminalEOF = createAction("event_terminal_eof", (data: string) => ({ payload: data }))
export const eClearTerminal = createAction("clear_terminal")
export const openTerminal = createAction("open_terminal", (el: HTMLElement) => ({ payload: el }))
export const closeTerminal = createAction("close_terminal")
export const cancelTerminal = createAction("cancel_terminal")

export default {
	openTerminal,
	closeTerminal,
}
