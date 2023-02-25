import React from 'react';
import ReactDOM from 'react-dom';

export function renderApp(children: React.ReactNode) {
	ReactDOM.render(
		<React.StrictMode>{children}</React.StrictMode>,
		document.getElementById('root'),
	);
}
