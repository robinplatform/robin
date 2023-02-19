import { Html, Head, Main, NextScript } from 'next/document';

export default function Document() {
	return (
		<Html>
			<Head>
				<link
					href={
						'https://cdn.jsdelivr.net/npm/monaco-editor@0.25.2/min/vs/editor/editor.main.css'
					}
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
