import styles from './sidebar.module.scss';
import React from 'react';
import Link from 'next/link';
import cx from 'classnames';
import { useRouter } from 'next/router';
import { ToolsIcon, HomeIcon } from '@primer/octicons-react';
import { useQuery } from '@tanstack/react-query';
import { toast } from 'react-hot-toast';
import { useRpcQuery } from '../hooks/useRpcQuery';
import { z } from 'zod';

type SidebarIcon = {
	icon: React.ReactNode;
	href: string;
	label: string;
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
			})
		),
		onError: err => {
			toast.error(`Failed to load robin apps: ${(err as Error).message}`);
		}
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
							className={cx(styles.primaryButton, 'robin-pad', {
								'robin-bg-dark-purple': href === router.asPath,
							})}
						>
							{icon}
						</Link>
						<span className={styles.sidebarLabel}>{label}</span>
					</div>
				))}
				{apps?.map(app => (
					<div key={app.name} className={styles.sidebarIconContainer}>
					<Link
						href={`/app/${app.id}`}
						className={cx(styles.primaryButton, 'robin-pad', {
							'robin-bg-dark-purple': window.location.pathname.startsWith(`/app/${app.id}`),
						})}
					>
						{app.pageIcon}
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
