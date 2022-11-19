import { useEffect, useState } from "react"
import { useAction, useSelect } from "~/store/hooks"

function Count() {
	const count = useSelect(state => state.app.count)
	const { timecount } = useAction().app
	useEffect(() => {
		timecount()
		return () => {
			timecount(false)
		}
	}, [timecount])
	return <span>{count}</span>
}

export default function Content() {
	const { command, cancel } = useAction().app
	const data = useSelect(state => state.app.data)
	const [enable, setEnable] = useState(false)
	return (
		<div>
			<button
				onClick={() => {
					command("seq", ["1", "5"])
					setEnable(true)
				}}
			>
				Submit
			</button>
			<button
				onClick={() => {
					cancel()
					setEnable(false)
				}}
			>
				Cancel
			</button>
			{enable && <Count />}
			<pre>{data.join("")}</pre>
		</div>
	)
}
