import { bindActionCreators, configureStore } from "@reduxjs/toolkit"
import { createContext, useMemo } from "react"
import { createDispatchHook, createSelectorHook, createStoreHook, ReactReduxContextValue } from "react-redux"
import createSagaMiddleware from "redux-saga"
import app from "~/store/app/action"
import rootSaga from "~/store/saga"
import terminal from "~/store/terminal/action"
import { initReducer, RootStore } from "./reducer"

export const AppStoreContext = createContext<ReactReduxContextValue<RootStore>>(null)
export const useStore = createStoreHook(AppStoreContext)
export const useDispatch = createDispatchHook(AppStoreContext)
export const useSelect = createSelectorHook(AppStoreContext)

export function useAction() {
	const dispatch = useDispatch()
	return useMemo(
		() => ({
			app: bindActionCreators(app, dispatch),
			terminal: bindActionCreators(terminal, dispatch),
		}),
		[dispatch],
	)
}

export function makeStore() {
	const sagaMiddleware = createSagaMiddleware()
	const store = configureStore({
		reducer: initReducer(),
		middleware: [sagaMiddleware],
		preloadedState: undefined,
		devTools: process.env.NODE_ENV === "development" ? { name: "react is awesome" } : false,
	})

	let saga = sagaMiddleware.run(rootSaga)

	if (module.hot) {
		module.hot.accept("./reducer", () => {
			// eslint-disable-next-line @typescript-eslint/no-var-requires
			store.replaceReducer(require("./reducer")(history))
		})

		module.hot.accept("./saga", () => {
			// eslint-disable-next-line @typescript-eslint/no-var-requires
			const root = require("./saga")
			saga?.cancel()
			saga = sagaMiddleware.run(root)
		})
	}

	return store
}

export const store = makeStore()
