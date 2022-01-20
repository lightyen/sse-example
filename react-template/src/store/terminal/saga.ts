import axios from "axios"
import { eventChannel, Task } from "redux-saga"
import { put, select, take, takeEvery, call, cancel } from "redux-saga/effects"
import { Terminal } from "xterm"
import { FitAddon } from "xterm-addon-fit"
import { WebglAddon } from "xterm-addon-webgl"
import { Unicode11Addon } from 'xterm-addon-unicode11'
import { establishEventStream } from "../app/action"

import "xterm/css/xterm.css"
import "xterm/lib/xterm.js"
import * as ac from "./action"
import { RootStore } from "../reducer"

function dataChannel(terminal: Terminal) {
	return eventChannel<string>(emit => {
		const d = terminal.onData(e => emit(e))
		return () => d.dispose()
	})
}

function binaryChannel(terminal: Terminal) {
	return eventChannel<string>(emit => {
		const d = terminal.onBinary(e => emit(e))
		return () => d.dispose()
	})
}

function copyToClipboardSync(text: string) {
	const el = document.createElement("textarea")
	el.style.position = "fixed"
	el.style.top = "0"
	el.style.left = "0"
	el.value = text
	try {
		document.body.appendChild(el)
		el.select()
		const ok = document.execCommand("copy")
		if (!ok) {
			console.error("Failed: copy text to clipboard")
		}
	} catch (err) {
		console.error("Failed: copy text to clipboard")
	} finally {
		document.body.removeChild(el)
	}
}

function attachCustomKey(terminal: Terminal) {
	return eventChannel<KeyboardEvent>(emit => {
		terminal.attachCustomKeyEventHandler(e => {
			const { key, code, ctrlKey, shiftKey, type } = e
			switch (key) {
				case "F5":
				case "F12":
					return false
			}

			if (key === "Escape") {
				if (type === "keydown") {
					emit(e)
				}
				return false
			}

			if (key === "Backspace") {
				if (type === "keydown") {
					emit(e)
				}
				return false
			}

			if (ctrlKey && shiftKey && code === "KeyC") {
				if (type === "keydown") {
					copyToClipboardSync(terminal.getSelection())
				}
				return false
			}

			return true
		})
		return () => void 0
	})
}

export default function* saga() {
	let terminal: Terminal | null = null
	let tasks: Task[]

	yield takeEvery(ac.command, function* ({ payload }) {
		try {
			yield call(axios.post, "/stream/terminal", { input: payload })
		} catch (e) {
			//
		}
	})

	yield takeEvery(ac.openTerminal, function* () {
		try {
			const source: EventSource | undefined = yield select((state: RootStore) => state.app.source)
			if (!source) {
				yield take(establishEventStream)
			}
			yield put(ac.command(""))
		} catch (e) {
			//
		}
	})

	yield takeEvery([ac.eTerminal, ac.eTerminalEOF, ac.eClearTerminal], function* (action) {
		if (terminal) {
			if (action.type === ac.eClearTerminal.type) {
				terminal.clear()
			} else {
				const data = action.payload.replace(/\x7f/g, "\b\x1b[P")
				if (data === "\x03") {
					// ^C
					yield put(ac.cancelTerminal())
				} else {
					terminal.write(data)
				}
			}
		}
	})
	yield takeEvery(ac.openTerminal, function* ({ payload: el }) {
		const term = new Terminal({
			fontFamily: `"Cascadia Code PL", "Cascadia Code", "ui-monospace", "Monaco","Consolas", "Liberation Mono", "Courier New", monospace`,
			cursorBlink: true,
			allowProposedApi: true,
			allowTransparency: false,
			rendererType: "canvas",
			convertEol: true,
			theme: {
				background: "#181818",
				cursor: "#eeeeee",
				cursorAccent: "#181818",
				black: "#181818",
				red: "#d12121",
			},
		})

		term.open(el)
		const fitAddon = new FitAddon()
		term.loadAddon(fitAddon)
		term.loadAddon(new Unicode11Addon())
		term.loadAddon(new WebglAddon())
		term.unicode.activeVersion = '11'
		fitAddon.fit()

		const data = dataChannel(term)
		const binary = binaryChannel(term)
		const customKey = attachCustomKey(term)

		tasks = [
			yield takeEvery(data, function* (data) {
				yield put(ac.command(data))
			}),
			yield takeEvery(binary, function* (data) {
				yield
				console.log("binary", data)
			}),
			yield takeEvery(customKey, function* ({ key }) {
				if (key === "Escape") {
					yield put(ac.command("\x1b"))
					return
				}

				if (key === "Backspace") {
					// yield put(ac.command("\x7f"))
					yield put(ac.command("\b"))
					return
				}
			}),
		]
		terminal = term
	})

	yield takeEvery(ac.closeTerminal, function* () {
		if (tasks) yield cancel(tasks)
		if (terminal) {
			yield put(ac.cancelTerminal())
			terminal.dispose()
			terminal = null
		}
	})

	yield takeEvery(ac.cancelTerminal, function* () {
		try {
			yield call(axios.post, "/stream/terminal/cancel")
		} catch {}
	})
}
