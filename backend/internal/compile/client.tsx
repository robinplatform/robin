// @ts-ignore
import { Page } from '__SCRIPT_PATH__';
import React from 'react';
import ReactDOM from 'react-dom';

class ErrorBoundary extends React.Component<
	React.PropsWithChildren,
	{ hasError: boolean }
> {
	constructor(props) {
		super(props);
		this.state = { hasError: false };
	}

	componentDidCatch(error, errorInfo) {
		console.log('OOOF YOU HAD AN ERROR', error);
		this.setState({ hasError: true });
	}

	render() {
		if (this.state.hasError) {
			return <h1>Something went wrong.</h1>;
		}

		return this.props.children;
	}
}

ReactDOM.render(
	<ErrorBoundary>
		<Page />
	</ErrorBoundary>,
	document.getElementById('root'),
);
