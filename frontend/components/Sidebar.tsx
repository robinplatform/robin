import styles from './sidebar.module.scss';
import React from 'react';
import Link from 'next/link';
import cx from 'classnames';
import { useRouter } from 'next/router';
import { ToolsIcon, HomeIcon, SyncIcon } from '@primer/octicons-react';
import { toast } from 'react-hot-toast';
import { useRpcMutation, useRpcQuery } from '../hooks/useRpcQuery';
import { z } from 'zod';
// @ts-ignore
import octicons from '@primer/octicons';

type SidebarIcon = {
	icon: React.ReactNode;
	href: string;
	label: string;
};

const AppIcon: React.FC<{ icon: string }> = ({ icon }) => {
	const svg = React.useMemo(() => octicons[icon]?.toSVG(), [icon]);
	if (svg) {
		return (
			<span
				style={{ fill: 'white' }}
				dangerouslySetInnerHTML={{ __html: svg }}
			/>
		);
	}
	return <>{icon}</>;
};

const RestartAppButton: React.FC = () => {
	const router = useRouter();
	const { mutate: restartApp, isLoading } = useRpcMutation({
		method: 'RestartApp',
		result: z.unknown(),

		onSuccess: () => {
			toast.success('Restarted app', { id: 'restart-app' });
		},
		onError: (err) => {
			toast.error(`Failed to restart app: ${(err as Error).message}`, {
				id: 'restart-app',
			});
		},
	});
	React.useEffect(() => {
		if (isLoading) {
			toast.loading('Restarting app', { id: 'restart-app' });
		}
	}, [isLoading]);

	if (router.pathname !== '/app/[id]') {
		return null;
	}

	return (
		<div className={cx(styles.homeWrapper, styles.sidebarIconContainer)}>
			<button
				type="button"
				disabled={isLoading}
				className={cx(styles.home, 'robin-rounded robin-bg-dark-purple')}
				onClick={() =>
					restartApp({
						appId: router.query.id as string,
					})
				}
			>
				<SyncIcon />
			</button>
			<span className={styles.sidebarLabel}>Restart app</span>
		</div>
	);
};

export function Sidebar() {
	const router = useRouter();

	const { data: apps } = useRpcQuery({
		method: 'GetApps',
		data: {},
		result: z.array(
			z.object({
				id: z.string(),
				name: z.string(),
				pageIcon: z.string(),
			}),
		),
		onError: (err) => {
			toast.error(`Failed to load robin apps: ${(err as Error).message}`);
		},
	});

	return (
		<div
			className={cx(styles.wrapper, 'col robin-bg-dark-blue robin-text-white')}
		>
			<div className="col">
				{apps?.map((app) => (
					<div key={app.name} className={styles.sidebarIconContainer}>
						<Link
							href={`/app/${app.id}`}
							className={cx(styles.sidebarLink, 'robin-pad', {
								'robin-bg-dark-purple': window.location.pathname.startsWith(
									`/app/${app.id}`,
								),
							})}
						>
							<span>
								<AppIcon icon={app.pageIcon} />
							</span>
						</Link>
						<span className={styles.sidebarLabel}>{app.name}</span>
					</div>
				))}
			</div>

			<div>
				<RestartAppButton />

				<div className={cx(styles.homeWrapper, styles.sidebarIconContainer)}>
					<Link
						href="/"
						className={cx(styles.home, 'robin-rounded robin-bg-dark-purple')}
					>
						<HomeIcon />
					</Link>
					<span className={styles.sidebarLabel}>Home</span>
				</div>

				<div className={cx(styles.homeWrapper, styles.sidebarIconContainer)}>
					<Link
						href="/settings"
						className={cx(styles.home, 'robin-rounded robin-bg-dark-purple')}
					>
						<ToolsIcon />
					</Link>
					<span className={styles.sidebarLabel}>Settings</span>
				</div>
			</div>
		</div>
	);
}
