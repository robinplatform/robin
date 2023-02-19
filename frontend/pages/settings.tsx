import Head from 'next/head';
import React from 'react';
import { Monaco } from '../components/Monaco';
import styles from './settings.module.css';
import cx from 'classnames';

export default function Settings() {
	const [showDiff, setShowDiff] = React.useState(false);

	// Reset updates when config is loaded
	const [configUpdates, setConfigUpdates] = React.useState<string>('{}');

	const onDiffEditorMount = React.useCallback(
		(editor: any) => {
			editor.getModifiedEditor().onDidChangeModelContent(function () {
				setConfigUpdates(editor.getModifiedEditor().getValue());
			});
		},
		[setConfigUpdates],
	);
	const onEditorMount = React.useCallback(
		(editor: any) => {
			editor.onDidChangeModelContent(function () {
				setConfigUpdates(editor.getValue());
			});
		},
		[setConfigUpdates],
	);

	return (
		<div className={'full robin-bg-dark-slate robin-pad'}>
			<Head>
				<title>Settings | Robin</title>
			</Head>

			<div
				className={'full col robin-gap robin-rounded robin-bg-slate robin-pad'}
			>
				<h1 className={'robin-text-bold robin-no-pad robin-text-xl'}>
					Settings
				</h1>

				<Monaco
					diffEditor={showDiff}
					language={'json'}
					defaultValue={'{}'}
					value={configUpdates}
					onMount={showDiff ? onDiffEditorMount : onEditorMount}
				/>

				<div className={styles.buttons}>
					<button onClick={() => setShowDiff(!showDiff)}>
						{showDiff ? 'Disable' : 'Enable'} diff view
					</button>

					<button onClick={() => {}}>Update config</button>
				</div>
			</div>
		</div>
	);
}
