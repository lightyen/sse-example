import { ThemeProvider } from "@emotion/react"
import { QueryClientProvider } from "@tanstack/react-query"
import { PropsWithChildren, StrictMode } from "react"
import { createRoot } from "react-dom/client"
import { LocaleProvider, StoreProvider } from "~/store/Provider"
import { App } from "./App"
import { GlobalStyles } from "./GlobalStyles"
import { useSelect } from "./store"

function ReactQueryProvider({ children }: PropsWithChildren<{}>) {
	const queryClient = useSelect(state => state.app.queryClient)
	return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
}

const rootEl = document.getElementById("root")

if (rootEl) {
	const root = createRoot(rootEl)
	root.render(
		<StrictMode>
			<GlobalStyles />
			<StoreProvider>
				<LocaleProvider>
					<ReactQueryProvider>
						<ThemeProvider theme={{ colors: { primary: "#abcaca9f" } }}>
							<App />
						</ThemeProvider>
					</ReactQueryProvider>
				</LocaleProvider>
			</StoreProvider>
		</StrictMode>,
	)
}
