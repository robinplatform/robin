import React from "react";
import { renderApp } from "@robinplatform/toolkit/react";
import { useRpcQuery } from "@robinplatform/toolkit/react/rpc";
import { getOsInfo } from "./app.server";

const App = () => {
	const { data: osInfo } = useRpcQuery(getOsInfo, {});
	if (!osInfo) {
		return <p>Loading...</p>;
	}

	return <p>You're on {osInfo?.platform}!</p>;
};

renderApp(<App />);
