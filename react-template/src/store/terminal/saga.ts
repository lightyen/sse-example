import { eventChannel } from "redux-saga"
import { put, take, takeEvery, fork } from "redux-saga/effects"
import { Terminal } from "xterm"
import { FitAddon } from "xterm-addon-fit"
import "xterm/css/xterm.css"
import "xterm/lib/xterm.js"
import * as ac from "./action"

export const term = new Terminal({
	fontFamily: "Cascadia Code PL",
	cursorBlink: true,
	allowProposedApi: true,
	allowTransparency: false,
	rendererType: "canvas",
	theme: {
		background: "#222222",
		cursor: "#eeeeee",
		cursorAccent: "#222222",
		black: "#222222",
	},
})

const printable_ascii = /^[\x20-\x7E]$/
let count = 0

const fitAddon = new FitAddon()
term.loadAddon(fitAddon)
// for (let i = 0xa0; i <= 0xd6; i++) {
// 	term.write(String.fromCharCode(0xe000 + i) + " ")
// }

export interface KeyEvent {
	key: string
	domEvent: KeyboardEvent
}

function key(terminal: Terminal) {
	return eventChannel(emit => {
		const d = terminal.onKey(e => emit(e))
		return () => d.dispose()
	})
}

function custom(terminal: Terminal) {
	return eventChannel(emit => {
		terminal.attachCustomKeyEventHandler(e => {
			const { key, ctrlKey, type } = e

			switch (key) {
				case "F5":
				case "F12":
				case "ArrowLeft":
				case "ArrowRight":
				case "ArrowUp":
				case "ArrowDown":
					return false
			}

			if (key === "Backspace") {
				if (type === "keydown") emit(e)
				return false
			}

			if (ctrlKey && key === "K") {
				if (type === "keydown") emit(e)
				return false
			}

			if (key === "Enter") {
				if (type === "keydown") emit(e)
				return false
			}

			return true
		})
		return () => void 0
	})
}

export default function* saga() {
	yield takeEvery(ac.eShell, function* ({ payload }) {
		yield
		if (payload === "\t") return
		term.write(payload)
		switch (payload) {
			case "\b":
				term.write("\x1b[P")
				break
		}
	})
	yield takeEvery(ac.openTerminal, function* () {
		fitAddon.fit()
		const ch = key(term)
		const customCh = custom(term)

		yield fork(function* () {
			while (true) {
				const { key, ctrlKey } = yield take(customCh)

				if (ctrlKey && key === "K") {
					term.clear()
					yield put(ac.shell(""))
					continue
				}

				if (key === "Backspace") {
					if (count > 0) {
						yield put(ac.shell("\b"))
						count--
					}
					continue
				}

				if (key === "Enter") {
					count = 0
					term.write("\n")
					yield put(ac.shell("\r"))
					continue
				}
			}
		})

		while (true) {
			const e: KeyEvent = yield take(ch)
			const { key } = e

			console.log(JSON.stringify({ key }))

			if (printable_ascii.test(key)) {
				count++
			}

			yield put(ac.shell(key))
		}
	})
}
