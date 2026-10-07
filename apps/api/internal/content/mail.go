package content

import (
	"context"
	"database/sql"
	"strings"

	"github.com/kapildev333/unicourse/apps/api/internal/outbox"
)

// announcementMails returns one mail per enrolled student and staff member of the section other than
// the author, ordered by email. Invitees without a user row get nothing (spec 6 section 7).
func announcementMails(ctx context.Context, tx *sql.Tx, baseURL, slug, sectionID, authorID, title, body string) ([]outbox.Mail, error) {
	var code, number, courseTitle, term, author string
	err := tx.QueryRowContext(ctx, `select c.code, s.number, c.title, t.name,
		(select coalesce(nullif(u.display_name, ''), u.email) from users u where u.id = $2)
		from sections s join courses c on c.id = s.course_id join terms t on t.id = s.term_id
		where s.id = $1`, sectionID, authorID).Scan(&code, &number, &courseTitle, &term, &author)
	if err != nil {
		return nil, err
	}
	rows, err := tx.QueryContext(ctx, `select u.email from users u
		where u.id in (select user_id from enrollments where section_id = $1
		               union select user_id from section_staff where section_id = $1)
		  and u.id <> $2
		  and exists (select 1 from memberships m where m.user_id = u.id and m.status = 'active')
		order by u.email`, sectionID, authorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	text := body + "\n\nPosted by " + author + " in " + code + " " + courseTitle + ", " + term + "\n" +
		strings.TrimRight(baseURL, "/") + "/" + slug + "/sections/" + sectionID + "/announcements\n"
	subject := "[" + code + "-" + number + "] " + title
	var mails []outbox.Mail
	for rows.Next() {
		var to string
		if err := rows.Scan(&to); err != nil {
			return nil, err
		}
		mails = append(mails, outbox.Mail{To: to, Subject: subject, Text: text})
	}
	return mails, rows.Err()
}
