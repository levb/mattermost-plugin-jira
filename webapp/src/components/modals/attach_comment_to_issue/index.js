// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {getPost} from 'mattermost-redux/selectors/entities/posts';
import {getCurrentTeam} from 'mattermost-redux/selectors/entities/teams';

import {closeAttachCommentToIssueModal, attachCommentToIssue} from 'actions';
import {isAttachCommentToIssueModalVisible, getAttachCommentToIssueModalForPostId} from 'selectors';

import AttachCommentToIssue from './attach_comment_to_issue';

const mapStateToProps = (state) => {
    const postId = getAttachCommentToIssueModalForPostId(state);
    const post = getPost(state, postId);
    const currentTeam = getCurrentTeam(state);

    return {
        visible: isAttachCommentToIssueModalVisible(state),
        post,
        currentTeam,
    };
};

const mapDispatchToProps = (dispatch) => bindActionCreators({
    close: closeAttachCommentToIssueModal,
    create: attachCommentToIssue,
}, dispatch);

export default connect(mapStateToProps, mapDispatchToProps)(AttachCommentToIssue);
