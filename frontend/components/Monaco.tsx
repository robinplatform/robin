import Editor, {
	DiffEditor,
	DiffEditorProps as MonacoDiffEditorProps,
	EditorProps,
} from '@monaco-editor/react';
import cx from 'classnames';
import * as React from 'react';
import styles from './monaco.module.css';

type DiffEditorProps = Omit<MonacoDiffEditorProps, 'original' | 'modified'> & {
	// Uses `defaultValue` as `original` in Monaco diff editor
	defaultValue: string;
	// Uses `value` as `modified` in Monaco diff editor
	value: string;
};

type MonacoProps = { disabled: boolean } & (
	| ({ diffEditor: true } & DiffEditorProps)
	| ({ diffEditor?: false } & EditorProps)
);

export const Monaco: React.FC<MonacoProps> = (props) => (
	<div className={cx(styles.editor, 'robin-border-white')}>
		<div className={cx('full', props.disabled && styles.disabled)}>
			{props.diffEditor ? (
				<DiffEditor
					theme={'vs-dark'}
					height={'100%'}
					original={props.defaultValue}
					modified={props.value}
					{...props}
				/>
			) : (
				<Editor
					theme={'vs-dark'}
					height={'100%'}
					language={'javascript'}
					{...props}
				/>
			)}
		</div>
	</div>
);
