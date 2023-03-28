import React from 'react';
import styles from './ScrollWindow.module.scss';
import cx from 'classnames';

type ScrollWindowProps = {
	className?: string;
	style?: React.CSSProperties;
	innerClassName?: string;
	innerStyle?: React.CSSProperties;
	children?: JSX.Element | JSX.Element[];
};

export const ScrollWindow = ({
	className,
	style,
	innerClassName,
	innerStyle,
	children,
}: ScrollWindowProps) => {
	/*  
    The position relative/absolute stuff makes it so that the
    inner div doesn't affect layout calculations of the surrounding div.
    I found this very confusing at first, so here's the SO post that I got it from:

    https://stackoverflow.com/questions/27433183/make-scrollable-div-take-up-remaining-height
    */
	return (
		<div className={cx(className, styles.wrapper)} style={style}>
			<div className={cx(innerClassName, styles.inner)} style={innerStyle}>
				{children}
			</div>
		</div>
	);
};
