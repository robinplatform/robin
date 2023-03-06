import React from 'react';
import { z } from 'zod';
import { useRpcQuery } from '../hooks/useRpcQuery';
import styles from './AppToolbar.module.scss';

export const AppToolbar: React.FC<{
	appId: string;
	actions: React.ReactNode;
}> = ({ appId, actions }) => {
	const { data: appConfig, error: errLoadingAppConfig } = useRpcQuery({
		method: 'GetAppById',
		data: { appId },
		result: z.object({
			id: z.string(),
			name: z.string(),
			pageIcon: z.string(),
		}),
	});

	return (
		<div className={styles.toolbar}>
			<>
				{!errLoadingAppConfig && <p>{appConfig?.name ?? 'Loading ...'}</p>}
				{errLoadingAppConfig && (
					<p>App failed to load: {String(errLoadingAppConfig)}</p>
				)}

				<div className={styles.toolbarIcons}>{actions}</div>
			</>
		</div>
	);
};
