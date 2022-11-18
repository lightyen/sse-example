import { createReducer } from "@reduxjs/toolkit"
import { QueryClient } from "@tanstack/react-query"
import * as ac from "./action"
import { AlertType } from "./alert"
import { getScreen, ScreenType } from "./screen"

const screen = getScreen()

function getMode(screen: ScreenType) {
	if (screen === "xs" || screen === "sm") return "mobile"
	if (screen === "md") return "tablet"
	return "desktop"
}

export interface AppStore {
	queryClient: QueryClient
	screen: ScreenType
	mode: "mobile" | "tablet" | "desktop"
	alert: {
		visible: boolean
		type: AlertType
		id?: string
		text?: string
	}
	count: number
	source?: EventSource
}

const init: AppStore = {
	queryClient: new QueryClient(),
	screen,
	mode: getMode(screen),
	alert: {
		visible: false,
		type: "info",
	},
	count: 0,
}

export const app = createReducer(init, builder =>
	builder
		.addCase(ac.eventStreamReady, (state, { payload }) => {
			state.source = payload
		})
		.addCase(ac.onScreenUpdated, (state, { payload: screen }) => {
			state.screen = screen
			state.mode = getMode(screen)
		})
		.addCase(ac.closeAlert, state => {
			state.alert.visible = false
		})
		.addCase(ac.newAlert, (state, { payload }) => {
			state.alert.visible = true
			state.alert.type = payload.type
			state.alert.text = payload.text
			state.alert.id = payload.id
		})
		.addCase(ac.eCount, (state, { payload }) => {
			state.count = Number(payload)
		}),
)
