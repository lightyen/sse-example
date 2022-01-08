import { Provider as ReactReduxProvider } from "react-redux"
import { AppStoreContext, store } from "~/store"
import View from "~/View"

function StoreProvider({ children }: { children?: React.ReactNode }) {
	return (
		<ReactReduxProvider context={AppStoreContext} store={store}>
			{children}
		</ReactReduxProvider>
	)
}

export default function App() {
	return (
		<StoreProvider>
			<View />
		</StoreProvider>
	)
}
