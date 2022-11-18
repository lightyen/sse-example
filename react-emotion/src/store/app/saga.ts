import { ActionCreatorWithoutPayload } from "@reduxjs/toolkit"
import axios from "axios"
import { Task, eventChannel } from "redux-saga"
import { call, cancel, cancelled, fork, put, select, take, takeEvery } from "redux-saga/effects"
import * as ac from "./action"
import { alert, autoCloseAlert } from "./alert"
import { ScreenType } from "./screen"

const mediaQuery = (query: string) =>
	eventChannel<MediaQueryListEvent>(emit => {
		const mql = window.matchMedia(query)
		function onchange(e: MediaQueryListEvent) {
			emit(e)
		}
		mql.addEventListener("change", onchange, { passive: true })
		return () => {
			mql.removeEventListener("change", onchange)
		}
	})

function screen(query: string, screen: ScreenType) {
	return fork(function* () {
		const ch = mediaQuery(query)
		while (true) {
			const event: MediaQueryListEvent = yield take(ch)
			if (event.matches) yield put(ac.onScreen({ event, screen }))
		}
	})
}

function onScreenUpdated() {
	return fork(function* () {
		while (true) {
			const e: ReturnType<typeof ac.onScreen> = yield take(ac.onScreen)
			yield put(ac.onScreenUpdated(e.payload.screen))
		}
	})
}

function event(source: EventSource, event: string) {
	return eventChannel(emit => {
		function callback(e: MessageEvent) {
			emit(e)
		}
		source.addEventListener(event, callback)
		return () => {
			source.removeEventListener(event, callback)
		}
	})
}

function eventStream(openAC: ActionCreatorWithoutPayload) {
	let task: Task | undefined
	return function* (action: ReturnType<ActionCreatorWithoutPayload>) {
		yield clear()
		if (action.type === openAC.type) {
			task = yield handleEventStream("/apis/stream")
		}
	}
	function* clear() {
		if (task) yield cancel(task)
		task = undefined
	}
}

function handleEventStream(url: string) {
	return fork(function* () {
		const source = new EventSource(url)

		yield fork(function* () {
			const ch = event(source, "error")
			while (true) {
				yield take(ch)
				yield put(ac.errorEventStream())
			}
		})

		let tasks: Task[] = []

		try {
			tasks = [
				yield fork(function* () {
					const ch = event(source, "timecount")
					while (true) {
						const { data } = yield take(ch)
						yield put(ac.eCount(data))
					}
				}),
			]

			const ch = event(source, "establish")
			while (true) {
				const { lastEventId } = yield take(ch)
				axios.defaults.headers.common["Last-Event-ID"] = lastEventId
				yield put(ac.eventStreamReady(source))
			}
		} finally {
			if (cancelled()) {
				yield cancel(tasks)
				source.close()
			}
		}
	})
}

export default function* app() {
	yield screen(`screen and (max-width: 639px)`, "xs")
	yield screen(`screen and (min-width: 640px) and (max-width: 767px)`, "sm")
	yield screen(`screen and (min-width: 768px) and (max-width: 1024px)`, "md")
	yield screen(`screen and (min-width: 1025px) and (max-width: 1279px)`, "lg")
	yield screen(`screen and (min-width: 1280px) and (max-width: 1535px)`, "xl")
	yield screen(`screen and (min-width: 1536px)`, "2xl")
	yield onScreenUpdated()

	yield takeEvery(ac.alert, function* ({ payload }) {
		yield alert(payload)
	})

	yield autoCloseAlert()

	yield takeEvery([ac.openEventStream, ac.closeEventStream], eventStream(ac.openEventStream))

	yield takeEvery(ac.timecount, function* ({ payload }) {
		try {
			if (payload === false) {
				yield call(axios.get, `/apis/timecount`, { params: { enable: "off" } })
			} else {
				const source: EventSource | undefined = yield select(state => state.app.source)
				if (!source) {
					yield take(ac.eventStreamReady)
				}
				yield call(axios.get, `/apis/timecount`)
			}
		} catch {}
	})

	yield put(ac.openEventStream())

	yield put(ac.timecount())
}
