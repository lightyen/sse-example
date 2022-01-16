import { createAction } from "@reduxjs/toolkit"

export const openEventStream = createAction("open_event_stream")
export const closeEventStream = createAction("close_event_stream")
export const establishEventStream = createAction<EventSource>("establish_event_stream")
export const errorEventStream = createAction("error_event_stream")

export const eCount = createAction("event_count", (data: string) => {
	return { payload: data }
})

export const timecount = createAction<false | undefined>("time_count")

export default {
	timecount,
}
