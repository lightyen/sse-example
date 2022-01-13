import { createAction } from "@reduxjs/toolkit"

export const openEventStream = createAction("open_event_stream")
export const closeEventStream = createAction("close_event_stream")
export const establishEventStream = createAction<EventSource>("establish_event_stream")
export const errorEventStream = createAction("error_event_stream")

export const command = createAction("COMMAND", (name: string, args: string[]) => {
	return {
		payload: {
			name,
			args,
		},
	}
})

export const eCommand = createAction("event_command", (payload: string) => ({ payload }))

export const eCount = createAction("event_count", (data: string) => {
	return { payload: data }
})

export const cancel = createAction("CANCEL")

export const timecount = createAction<false | undefined>("TIME_COUNT")

export default {
	command,
	cancel,
	timecount,
}
