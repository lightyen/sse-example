import { createAction } from "@reduxjs/toolkit"
import { nomalizeAlertOption, WeakAlertOption } from "./alert"

export const onScreen = createAction<{ event: MediaQueryListEvent; screen: "2xl" | "xl" | "lg" | "md" | "sm" | "xs" }>(
	"on_screen",
)
export const onScreenUpdated = createAction<"2xl" | "xl" | "lg" | "md" | "sm" | "xs">("on_screen_updated")

export const alert = createAction<WeakAlertOption>("alert")
export const closeAlert = createAction("closeAlert")
export const newAlert = createAction("newAlert", (option: WeakAlertOption) => {
	return { payload: nomalizeAlertOption(option) }
})

export const openEventStream = createAction("open_event_stream")
export const closeEventStream = createAction("close_event_stream")
export const errorEventStream = createAction("error_event_stream")
export const eventStreamReady = createAction<EventSource>("event_stream_ready")

export const eCount = createAction("event_count", (data: string) => ({ payload: data }))

export const timecount = createAction<false | undefined>("timecount")

export default {
	alert,
	closeAlert,
}
