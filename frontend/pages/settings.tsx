import Head from 'next/head';
import React from 'react';
import { Monaco } from '../components/Monaco';
import styles from './settings.module.scss';
import { getConfig, updateConfig } from '@robinplatform/toolkit';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useMonaco } from '@monaco-editor/react';
import { Button } from '../components/Button';
import { FileMovedIcon, FileCodeIcon } from '@primer/octicons-react';
import toast from 'react-hot-toast';
import { Alert } from '../components/Alert';
import { Spinner } from '../components/Spinner';
import cx from 'classnames';

export default function Settings() {
	const {
		data: config,
		isLoading,
		error,
	} = useQuery({
		queryKey: ['getConfig'],
		queryFn: getConfig,
		refetchOnWindowFocus: false,
	});
	const configJson = React.useMemo(
		() => JSON.stringify(config, null, '\t'),
		[config],
	);

	const [showDiff, setShowDiff] = React.useState(false);

	// Reset updates when config is loaded
	const [configUpdates, setConfigUpdates] = React.useState<string>('{}');
	React.useEffect(() => {
		if (configJson) {
			setConfigUpdates(configJson);
		}
	}, [configJson]);

	const queryClient = useQueryClient();
	const { mutate: performUpdate, isLoading: isUpdating } = useMutation({
		mutationFn: updateConfig,
		onSuccess: () => {
			queryClient.invalidateQueries(['getConfig']);
			toast.success('Settings updated successfully');
		},
		onError: () => {
			toast.error('Failed to update settings');
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
		<div className={'full robin-bg-light-blue robin-pad'}>
			<Head>
				<title>Settings | Robin</title>
			</Head>

			<div
				className={
					'full col robin-gap robin-rounded robin-bg-dark-blue robin-pad'
				}
			>
				<h1
					className={cx(
						styles.title,
						'robin-text-bold robin-no-pad robin-text-xl',
					)}
				>
					<span>Settings</span>
					{isLoading && (
						<span style={{ marginLeft: '1rem' }}>
							<Spinner />
						</span>
					)}
				</h1>

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
					disabled={isLoading || isUpdating}
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
						onClick={() => performUpdate(configUpdates)}
					>
						Update config
					</Button>
				</div>
			</div>
		</div>
	);
}
