import React from 'react';
import styles from './ScrollWindow.module.scss';
import cx from 'classnames';

type ScrollWindowProps = {
	className?: string;
	style?: React.CSSProperties;
	innerClassName?: string;
	innerStyle?: React.CSSProperties;
	children?: React.ReactNode;
	scrollStyle: React.CSSProperties['overflowY'];
};

export const ScrollWindow = ({
	className,
	style,
	innerClassName,
	scrollStyle = 'scroll',
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
			<div className={styles.inner} style={{ overflowY: scrollStyle }}>
				{/* Maybe innerClassName and innerStyle aren't necessary, but I like them as a way to reduce
					the indentation level of the caller's children */}
				<div className={innerClassName} style={innerStyle}>
					{children}
				</div>
			</div>
		</div>
	);
};
