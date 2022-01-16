import { createReducer } from "@reduxjs/toolkit"
import "xterm/css/xterm.css"
import "xterm/lib/xterm.js"
import * as ac from "./action"

export interface AppStore {
	source?: EventSource | undefined
	count?: string
}

const init: AppStore = {}

export const app = createReducer(init, builder =>
	builder
		.addCase(ac.establishEventStream, (state, { payload }) => {
			state.source = payload
		})
		.addCase(ac.closeEventStream, state => {
			state.source = undefined
		})
		.addCase(ac.eCount, (state, { payload }) => {
			state.count = payload
		}),
)
