import styles from './Sidebar.module.scss';
import React from 'react';
import Link from 'next/link';
import cx from 'classnames';
import { HomeIcon, GearIcon } from '@primer/octicons-react';
import { toast } from 'react-hot-toast';
import { useRpcQuery } from '../hooks/useRpcQuery';
import { z } from 'zod';
// @ts-ignore
import octicons from '@primer/octicons';

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

export function Sidebar() {
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
						<GearIcon />
					</Link>
					<span className={styles.sidebarLabel}>Settings</span>
				</div>
			</div>
		</div>
	);
}
