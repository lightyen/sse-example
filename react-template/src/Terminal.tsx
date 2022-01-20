import { useEffect, useRef } from "react"
import { useAction } from "~/store"

export function Terminal() {
	const { openTerminal, closeTerminal } = useAction().terminal
	const ref = useRef<HTMLDivElement>()
	useEffect(() => {
		openTerminal(ref.current)
		return () => {
			closeTerminal()
		}
	}, [openTerminal, closeTerminal])
	return (
		<div style={{ padding: "2rem", width: "800px" }}>
			<div ref={ref} />
		</div>
	)
}
