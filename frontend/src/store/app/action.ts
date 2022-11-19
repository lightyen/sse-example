import { createAction } from "@reduxjs/toolkit"

export const openEventStream = createAction("open_event_stream")
export const closeEventStream = createAction("close_event_stream")
export const establishEventStream = createAction<EventSource>("establish_event_stream")
export const errorEventStream = createAction("error_event_stream")

export const command = createAction("command", (name: string, args: string[]) => {
	return {
		payload: {
			name,
			args,
		},
	}
})

export const eCommand = createAction("event_command", (payload: string) => ({ payload }))

export const eCount = createAction("event_count", (data: string) => ({ payload: data }))

export const commandCancel = createAction("command_cancel")

export const timecount = createAction<false | undefined>("timecount")

export default {
	command,
	cancel: commandCancel,
	timecount,
}
