import { combineReducers } from "@reduxjs/toolkit"
import { app, AppStore } from "./app/reducer"

interface RootStoreType {
	app: AppStore
}

type DeepReadonly<T> = {
	readonly [K in keyof T]: T[K] extends Record<string, unknown> ? DeepReadonly<T[K]> : T[K]
}

export type RootStore = DeepReadonly<RootStoreType>

export function initReducer() {
	return combineReducers({
		app,
	})
}
