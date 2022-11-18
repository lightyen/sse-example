import { tw } from "twobj"
import { useSelect } from "./store"

export function App() {
	return (
		<div
			tw="flex flex-col items-center justify-center h-screen to-pink-400"
			css={[tw`bg-gradient-to-b from-blue-900 to-blue-400 content-around`]}
		>
			<Counter />
		</div>
	)
}

function Counter() {
	const count = useSelect(state => state.app.count)
	return <div tw="text-4xl">{count}</div>
}
