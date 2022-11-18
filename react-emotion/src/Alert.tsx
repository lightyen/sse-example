import { AnimatePresence, motion } from "framer-motion"
import { FormattedMessage } from "react-intl"
import { useSelect } from "./store"

export function Alert() {
	const { visible } = useSelect(state => state.app.alert)
	return <AnimatePresence>{visible && <AlertPopper />}</AnimatePresence>
}

function AlertPopper() {
	// const intl = useIntl()
	// const { closeAlert } = useAction().app
	const { type, id, text } = useSelect(state => state.app.alert)
	return (
		<motion.div
			initial="initial"
			animate="animate"
			exit="exit"
			id="app-alert"
			transition={{ type: "spring", damping: 50, stiffness: 600 }}
			variants={{
				initial: { opacity: 0, x: "0%", y: "100%" },
				animate: { opacity: 1, x: "0%", y: 0 },
				exit: { opacity: 0, x: "0%", y: "150%" },
			}}
			tw="w-auto z-[9999] transform translate-y-full
			fixed [inset: auto auto 1rem 0]"
		>
			<div tw="m-6 p-3 bg-red-300 rounded-lg">
				<ul tw="">
					<li>type: {type}</li>
					<li>id: {id}</li>
					<li>text: {text || <FormattedMessage id={id} />}</li>
				</ul>
			</div>
		</motion.div>
	)
}
