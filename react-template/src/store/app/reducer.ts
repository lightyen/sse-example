import { createReducer } from "@reduxjs/toolkit"
import * as ac from "./action"

export interface AppStore {
	source?: EventSource | undefined
	lastEventId: number
	data: string[]
	count?: string
}

const init: AppStore = {
	lastEventId: 0,
	data: [],
}

export const app = createReducer(init, builder =>
	builder
		.addCase(ac.establishEventStream, (state, { payload }) => {
			state.source = payload
		})
		.addCase(ac.closeEventStream, state => {
			state.source = undefined
		})
		.addCase(ac.eCommand, (state, { payload }) => {
			const id = parseInt(payload.id)
			if (state.lastEventId < id) {
				state.data = [payload.data]
				state.lastEventId = id
			} else if (state.lastEventId === id) {
				state.data = state.data.concat(payload.data)
			}
		})
		.addCase(ac.eCount, (state, { payload }) => {
			state.count = payload
		}),
)
