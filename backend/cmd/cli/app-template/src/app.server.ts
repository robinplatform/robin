import * as os from "os";

export async function getOsInfo() {
	return {
		platform: os.platform(),
	};
}
