import { Html, Head, Main, NextScript } from 'next/document';

export default function Document() {
	return (
		<Html>
			<Head>
				{/* This uses a static local copy because otherwise the app gets hyper-slowed by web requests */}
				<link
					href={'/monaco-editor.css'}
					data-name={'vs/editor/editor.main'}
					type={'text/css'}
					rel={'stylesheet'}
				/>
			</Head>

			<body>
				<Main />
				<NextScript />
			</body>
		</Html>
	);
}
