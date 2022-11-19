import View from "~/View"
import { StoreProvider } from "./store/Provider"

export default function App() {
	return (
		<StoreProvider>
			<View />
		</StoreProvider>
	)
}
