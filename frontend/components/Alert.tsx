import styles from './Alert.module.scss';
import cx from 'classnames';

export type AlertProps = React.PropsWithChildren<{
	variant: 'error';
	title: string;
}>;

export const Alert: React.FC<AlertProps> = ({ variant, title, children }) => {
	return (
		<div
			className={cx(styles.alertContainer, {
				[styles.alertError]: variant === 'error',
			})}
		>
			<p className={styles.alertTitle}>{title}</p>
			{children}
		</div>
	);
};
