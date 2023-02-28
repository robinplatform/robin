import React from 'react';
import cx from 'classnames';

import styles from './Button.module.scss';

export type ButtonProps = React.PropsWithChildren<{
	variant: 'primary' | 'secondary';
	size?: 'sm' | 'md' | 'lg';
	isLoading?: boolean;
	icon?: React.ReactNode;
	onClick(): void;
}>;

export const Button: React.FC<ButtonProps> = ({
	size,
	variant,
	isLoading,
	icon,
	onClick,
	children,
}) => {
	return (
		<button
			type="button"
			onClick={onClick}
			disabled={isLoading}
			className={cx({
				[styles.btnPrimary]: variant === 'primary',
				[styles.btnSecondary]: variant === 'secondary',

				[styles.btnSm]: size === 'sm',
				[styles.btnMd]: size === 'md' || !size,
				[styles.btnLg]: size === 'lg',
			})}
		>
			{icon && <span className={styles.icon}>{icon}</span>}
			<span>{children}</span>
		</button>
	);
};
