import { fork } from "redux-saga/effects"
import app from "./app/saga"
import terminal from "./terminal/saga"

export default function* root() {
	yield fork(app)
	yield fork(terminal)
}
