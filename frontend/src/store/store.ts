import { configureStore } from "@reduxjs/toolkit"
import createSagaMiddleware from "redux-saga"
import { app, AppStore } from "./app/reducer"
import rootSaga from "./saga"

// https://vitejs.dev/guide/env-and-mode.html#env-files

interface RootStoreType {
	app: AppStore
}

export type RootStore = Readonly<RootStoreType>

export function makeStore() {
	const sagaMiddleware = createSagaMiddleware()
	const store = configureStore({
		reducer: {
			app,
		},
		middleware: [sagaMiddleware],
		devTools: import.meta.env.MODE === "development" ? { name: import.meta.env.VITE_APP_NAME } : false,
	})

	sagaMiddleware.run(rootSaga)
	return store
}

export const store = makeStore()
