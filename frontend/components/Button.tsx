import React from 'react';
import cx from 'classnames';

import styles from './Button.module.scss';

export type ButtonProps = React.PropsWithChildren<{
	variant: 'primary' | 'secondary';
	isLoading?: boolean;
	icon?: React.ReactNode;
	onClick(): void;
}>;

export const Button: React.FC<ButtonProps> = ({
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
			})}
		>
			{icon && <span className={styles.icon}>{icon}</span>}
			<span>{children}</span>
		</button>
	);
};
