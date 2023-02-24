import styles from './sidebar.module.scss';
import React from 'react';
import Link from 'next/link';
import cx from 'classnames';
import { useRouter } from 'next/router';
import { ToolsIcon, HomeIcon } from '@primer/octicons-react';
import { toast } from 'react-hot-toast';
import { useRpcQuery } from '../hooks/useRpcQuery';
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

export function Sidebar() {
	const router = useRouter();
	const icons = React.useMemo<SidebarIcon[]>(
		() => [
			{
				icon: <ToolsIcon />,
				href: '/settings',
				label: 'Settings',
			},
		],
		[],
	);

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
				{icons.map(({ icon, href, label }) => (
					<div key={href} className={styles.sidebarIconContainer}>
						<Link
							href={href}
							className={cx(styles.sidebarLink, 'robin-pad', {
								'robin-bg-dark-purple': href === router.asPath,
							})}
						>
							{icon}
						</Link>
						<span className={styles.sidebarLabel}>{label}</span>
					</div>
				))}
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

			<div className={cx(styles.homeWrapper, styles.sidebarIconContainer)}>
				<Link
					href="/"
					className={cx(styles.home, 'robin-rounded robin-bg-dark-purple')}
				>
					<HomeIcon />
				</Link>
				<span className={styles.sidebarLabel}>Home</span>
			</div>
		</div>
	);
}
