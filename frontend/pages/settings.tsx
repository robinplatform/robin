import Head from 'next/head';
import React from 'react';
import { Monaco } from '../components/Monaco';
import styles from './settings.module.css';
import { getConfig, updateConfig } from '@robin/toolkit';
import { useMutation, useQuery, useQueryClient } from 'react-query';
import { useMonaco } from '@monaco-editor/react';

export default function Settings() {
	const { data: config, isLoading } = useQuery({
		queryKey: ['getConfig'],
		queryFn: getConfig,
		onSuccess: (res) => {},
	});
	const configJson = React.useMemo(
		() => JSON.stringify(config, null, '\t'),
		[config],
	);

	const [showDiff, setShowDiff] = React.useState(false);

	// Reset updates when config is loaded
	const [configUpdates, setConfigUpdates] = React.useState<string>('{}');
	React.useEffect(() => {
		if (configJson) setConfigUpdates(configJson);
	}, [configJson]);

	const queryClient = useQueryClient();
	const { mutate: performUpdate, isLoading: isUpdating } = useMutation({
		mutationFn: updateConfig,
		onSuccess: () => {
			queryClient.invalidateQueries(['getConfig']);
			console.log('Settings updated successfully');
		},
	});

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
					disabled={isLoading || isUpdating}
				/>

				<div className={styles.buttons}>
					<button onClick={() => setShowDiff(!showDiff)}>
						{showDiff ? 'Disable' : 'Enable'} diff view
					</button>

					<button onClick={() => performUpdate(configUpdates)}>
						Update config
					</button>
				</div>
			</div>
		</div>
	);
}
