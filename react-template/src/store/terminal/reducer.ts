import { createReducer } from "@reduxjs/toolkit"
import { Terminal } from "xterm"
import * as ac from "./action"
import { term } from "./saga"
export interface TerminalStore {
	readonly term: Terminal
	running: boolean
	history: string[]
}

const init: TerminalStore = {
	term,
	running: false,
	history: [],
}

export const terminal = createReducer(init, builder =>
	builder
		.addCase(ac.openTerminal, (state, { payload }) => {
			state.term.open(payload)
		})
		.addCase(ac.closeTerminal, state => {
			state.term.dispose()
		})
		.addCase(ac.addCommandHistory, (state, { payload }) => {
			state.history = state.history.concat(payload)
		})
		.addCase(ac.command, state => {
			state.running = true
		})
		.addCase(ac.eCommandEOF, state => {
			state.running = false
		}),
)
