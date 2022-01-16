import { createReducer } from "@reduxjs/toolkit"
import { Terminal } from "xterm"
import * as ac from "./action"
import { term } from "./saga"
export interface TerminalStore {
	readonly term: Terminal
}

const init: TerminalStore = {
	term,
}

export const terminal = createReducer(init, builder =>
	builder
		.addCase(ac.openTerminal, (state, { payload }) => {
			state.term.open(payload)
		})
		.addCase(ac.closeTerminal, state => {
			state.term.dispose()
		}),
)
