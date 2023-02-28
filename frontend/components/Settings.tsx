import Head from 'next/head';
import React from 'react';
import { Monaco } from './Monaco';
import styles from './Settings.module.scss';
import { useMonaco } from '@monaco-editor/react';
import { Button } from './Button';
import { FileMovedIcon, FileCodeIcon } from '@primer/octicons-react';
import { Alert } from './Alert';
import { Spinner } from './Spinner';
import cx from 'classnames';
import { z } from 'zod';

export interface SettingsProps<Shape> {
	title?: string;
	// TODO: validate schema and show the errors
	schema: z.ZodSchema<Shape>;
	isLoading: boolean;
	error?: string;
	value: Shape;
	onChange: (value: Shape) => void;
}

export function Settings<Shape>({
	title,
	isLoading,
	error,
	value,
	onChange,
}: SettingsProps<Shape>) {
	const configJson = React.useMemo(
		() => JSON.stringify(value, null, '\t'),
		[value],
	);

	const [showDiff, setShowDiff] = React.useState(false);

	// Reset updates when config is loaded
	const [configUpdates, setConfigUpdates] = React.useState<string>('{}');
	React.useEffect(() => {
		if (configJson) {
			setConfigUpdates(configJson);
		}
	}, [configJson]);

	const monaco = useMonaco();
	React.useEffect(() => {
		if (monaco) {
			monaco.languages.json.jsonDefaults.setDiagnosticsOptions({
				allowComments: true,
			});
		}
	}, [monaco]);

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
		<div className={'full'}>
			<div className={'full col robin-rounded robin-bg-dark-blue robin-pad'}>
				{title && (
					<h1
						className={cx(
							styles.title,
							'robin-text-bold robin-no-pad robin-text-xl',
						)}
					>
						<span>{title}</span>
						{isLoading && (
							<span style={{ marginLeft: '1rem' }}>
								<Spinner />
							</span>
						)}
					</h1>
				)}

				{!!error && (
					<Alert variant="error" title="Failed to load settings">
						<pre>
							<code>{String(error)}</code>
						</pre>
					</Alert>
				)}

				<Monaco
					diffEditor={showDiff}
					language={'json'}
					defaultValue={'{}'}
					value={configUpdates}
					onMount={showDiff ? onDiffEditorMount : onEditorMount}
					disabled={isLoading}
				/>

				<div className={styles.buttons}>
					<Button
						variant="secondary"
						icon={<FileCodeIcon />}
						onClick={() => setShowDiff(!showDiff)}
					>
						{showDiff ? 'Disable' : 'Enable'} diff view
					</Button>

					<Button
						variant="primary"
						icon={<FileMovedIcon />}
						onClick={() => onChange(JSON.parse(configUpdates))}
					>
						Update config
					</Button>
				</div>
			</div>
		</div>
	);
}
