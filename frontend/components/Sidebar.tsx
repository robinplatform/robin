import styles from './sidebar.module.css';
import React from 'react';
import Link from 'next/link';
import cx from 'classnames';
import { useRouter } from 'next/router';
import { ToolsIcon, HomeIcon } from '@primer/octicons-react';
import Tooltip from '@tippyjs/react';

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
				icon: <ToolsIcon size={24} />,
				href: '/settings',
				label: 'Settings',
			},
		],
		[],
	);

	return (
		<div className={cx(styles.wrapper, 'col robin-bg-slate robin-text-white')}>
			<div className="col">
				{icons.map(({ icon, href, label }) => (
					<Tooltip content={label} placement={'right'}>
						<Link
							href={href}
							className={cx(
								styles.primaryButton,
								href === router.asPath && 'robin-bg-green',
								'robin-pad',
							)}
						>
							{icon}
						</Link>
					</Tooltip>
				))}
			</div>

			<Tooltip content={'Home'} placement={'top'}>
				<div className={styles.homeWrapper}>
					<Link
						href="/"
						className={cx(styles.home, 'robin-rounded robin-bg-dark-slate')}
					>
						<HomeIcon />
					</Link>
				</div>
			</Tooltip>
		</div>
	);
}
