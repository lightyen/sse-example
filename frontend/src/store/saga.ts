import { fork } from "redux-saga/effects"
import app from "./app/saga"

export default function* root() {
	yield fork(app)
}
