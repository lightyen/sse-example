import { createContext, useMemo } from "react"
import {
	createDispatchHook,
	createSelectorHook,
	createStoreHook,
	ReactReduxContextValue,
	TypedUseSelectorHook,
} from "react-redux"
import { bindActionCreators } from "redux"
import app from "./app/action"
import { RootStore } from "./store"

export const AppStoreContext = createContext(null as unknown as ReactReduxContextValue<RootStore>)
export const useStore = createStoreHook(AppStoreContext)
export const useDispatch = createDispatchHook(AppStoreContext)
export const useSelect: TypedUseSelectorHook<RootStore> = createSelectorHook(AppStoreContext)
export function useAction() {
	const dispatch = useDispatch()
	return useMemo(
		() => ({
			app: bindActionCreators(app, dispatch),
		}),
		[dispatch],
	)
}
